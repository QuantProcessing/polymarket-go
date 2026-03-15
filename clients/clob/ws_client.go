package clob

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocket endpoints
const (
	WsURL = "wss://ws-subscriptions-clob.polymarket.com/ws"
)

// WebSocket channel types
const (
	WsChannelMarket = "market"
	WsChannelBook   = "book"
	WsChannelTrade  = "trade"
	WsChannelUser   = "user"
)

// storedSubscription stores the full subscription payload + handler for replay after reconnect
type storedSubscription struct {
	Key     string
	Payload WsSubscription
	Handler WsHandler
}

// WsClient is a standalone WebSocket client for Polymarket
// Users create it independently: ws := polymarket.NewWsClient(...)
type WsClient struct {
	baseURL     string
	channelType string // "market" or "user"
	logger      *zap.SugaredLogger

	conn   *websocket.Conn
	connMu sync.RWMutex

	// Subscription handlers (channelKey -> handler)
	handlers   map[string]WsHandler
	handlersMu sync.RWMutex

	// Stored subscriptions for replay after reconnect
	storedSubs   []storedSubscription
	storedSubsMu sync.Mutex

	// Control channels
	done      chan struct{} // closed to signal goroutines to stop
	reconnect chan struct{} // trigger reconnection

	// Read/write control
	writeMu sync.Mutex
}

// WsHandler is a callback function for WebSocket messages
type WsHandler func(data []byte) error

// WsSubscription represents a subscription request
type WsSubscription struct {
	Type                 string        `json:"type,omitempty"`      // Channel type: "market" or "user"
	Operation            string        `json:"operation,omitempty"` // "subscribe" or "unsubscribe" for dynamic ops
	Auth                 *WsAuthParams `json:"auth,omitempty"`
	Markets              []string      `json:"markets,omitempty"`                // For user channel
	AssetsIDs            []string      `json:"assets_ids,omitempty"`             // For market channel
	CustomFeatureEnabled bool          `json:"custom_feature_enabled,omitempty"` // For market channel
}

// WsAuthParams for authenticated channels
type WsAuthParams struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// WsMessage represents incoming WebSocket messages
type WsMessage struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// NewWsClient creates a new standalone WebSocket client
// baseURL should be the base WebSocket URL without the channel path
func NewWsClient(baseURL string, logger *zap.SugaredLogger) *WsClient {
	if baseURL == "" {
		baseURL = "wss://ws-subscriptions-clob.polymarket.com"
	}

	return &WsClient{
		baseURL:    baseURL,
		logger:     logger,
		handlers:   make(map[string]WsHandler),
		storedSubs: make([]storedSubscription, 0),
		done:       make(chan struct{}),
		reconnect:  make(chan struct{}, 1),
	}
}

// Connect establishes WebSocket connection to a specific channel
// channelType should be "market" or "user"
func (ws *WsClient) Connect(ctx context.Context, channelType string) error {
	ws.connMu.Lock()
	defer ws.connMu.Unlock()

	if ws.conn != nil {
		return fmt.Errorf("already connected")
	}

	// Validate channel type
	if channelType != WsChannelMarket && channelType != WsChannelUser {
		return fmt.Errorf("invalid channel type: %s (must be 'market' or 'user')", channelType)
	}

	ws.channelType = channelType

	if err := ws.dialAndStart(ctx); err != nil {
		return err
	}

	// Start reconnection handler (only once, on initial Connect)
	go ws.handleReconnect(ctx)

	return nil
}

// dialAndStart dials the WebSocket and starts read/ping goroutines.
// Must be called with connMu held.
func (ws *WsClient) dialAndStart(ctx context.Context) error {
	// Build channel-specific URL: wss://ws-subscriptions-clob.polymarket.com/ws/market
	url := fmt.Sprintf("%s/ws/%s", ws.baseURL, ws.channelType)
	ws.logger.Infow("Connecting to WebSocket", "url", url, "channel", ws.channelType)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	ws.conn = conn
	ws.logger.Info("WebSocket connected")

	// Re-create done channel for new goroutines
	ws.done = make(chan struct{})

	// Start message reader
	go ws.readMessages()

	// Start PING routine (client sends PING every 10s to keep alive)
	go ws.startPingRoutine()

	return nil
}

// Subscribe sends initial subscription message or dynamic subscription
// For initial subscription (right after Connect), set sub.Type to channel type
// For dynamic subscription, set sub.Operation to "subscribe"
func (ws *WsClient) Subscribe(sub WsSubscription, handler WsHandler) error {
	ws.connMu.RLock()
	if ws.conn == nil {
		ws.connMu.RUnlock()
		return fmt.Errorf("not connected")
	}
	channelType := ws.channelType
	ws.connMu.RUnlock()

	// Determine if this is initial or dynamic subscription
	isInitial := sub.Type != "" && sub.Operation == ""
	isDynamic := sub.Operation != ""

	// If neither is set, default to dynamic subscription
	if !isInitial && !isDynamic {
		sub.Operation = "subscribe"
		isDynamic = true
	}

	// Store handler based on channel and identifiers
	var channelKey string
	if len(sub.AssetsIDs) > 0 {
		channelKey = fmt.Sprintf("%s:%s", channelType, sub.AssetsIDs[0])
	} else if len(sub.Markets) > 0 {
		channelKey = fmt.Sprintf("%s:%s", channelType, sub.Markets[0])
	} else {
		channelKey = channelType
	}

	if handler != nil {
		ws.handlersMu.Lock()
		ws.handlers[channelKey] = handler
		ws.handlersMu.Unlock()
	}

	// Send subscription message
	ws.writeMu.Lock()
	err := ws.conn.WriteJSON(sub)
	ws.writeMu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	if isInitial {
		ws.logger.Infow("Sent initial subscription", "type", sub.Type, "key", channelKey)
	} else {
		ws.logger.Infow("Sent dynamic subscription", "operation", sub.Operation, "key", channelKey)
	}
	return nil
}

// Unsubscribe from specific assets/markets
// assetsIDs: list of asset IDs to unsubscribe from (for market channel)
func (ws *WsClient) Unsubscribe(assetsIDs []string) error {
	ws.connMu.RLock()
	if ws.conn == nil {
		ws.connMu.RUnlock()
		return fmt.Errorf("not connected")
	}
	channelType := ws.channelType
	ws.connMu.RUnlock()

	// Build unsubscribe message
	unsub := WsSubscription{
		Operation: "unsubscribe",
	}

	if channelType == WsChannelMarket {
		unsub.AssetsIDs = assetsIDs
	} else {
		// For user channel, use markets field (if supported)
		unsub.Markets = assetsIDs
	}

	ws.writeMu.Lock()
	err := ws.conn.WriteJSON(unsub)
	ws.writeMu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	// Remove handlers for these assets
	ws.handlersMu.Lock()
	for _, assetID := range assetsIDs {
		channelKey := fmt.Sprintf("%s:%s", channelType, assetID)
		delete(ws.handlers, channelKey)
	}
	ws.handlersMu.Unlock()

	ws.logger.Infow("Unsubscribed from assets", "assets", assetsIDs, "channel", channelType)
	return nil
}

// Close closes the WebSocket connection
func (ws *WsClient) Close() error {
	ws.connMu.Lock()
	defer ws.connMu.Unlock()

	// If already closed, return early
	if ws.conn == nil {
		return nil
	}

	// Close done channel to signal goroutines to stop
	select {
	case <-ws.done:
		// Already closed
	default:
		close(ws.done)
	}

	err := ws.conn.Close()
	ws.conn = nil
	return err
}

// startPingRoutine sends PING messages every 10 seconds to keep connection alive
// Polymarket expects clients to send PING every ~10s on market/user channels.
func (ws *WsClient) startPingRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ws.done:
			ws.logger.Debug("PING routine stopped")
			return
		case <-ticker.C:
			ws.connMu.RLock()
			conn := ws.conn
			ws.connMu.RUnlock()

			if conn == nil {
				return
			}

			ws.writeMu.Lock()
			err := conn.WriteMessage(websocket.TextMessage, []byte("PING"))
			ws.writeMu.Unlock()

			if err != nil {
				ws.logger.Warnw("Failed to send PING", "error", err)
				// Trigger reconnection
				ws.triggerReconnect()
				return
			}

			ws.logger.Debug("Sent PING")
		}
	}
}

// readMessages continuously reads messages from WebSocket
func (ws *WsClient) readMessages() {
	defer func() {
		if r := recover(); r != nil {
			ws.logger.Errorw("Panic in readMessages", "panic", r)
		}
	}()

	for {
		select {
		case <-ws.done:
			return
		default:
		}

		ws.connMu.RLock()
		conn := ws.conn
		ws.connMu.RUnlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// Only log if not intentionally closed
			select {
			case <-ws.done:
				return
			default:
			}

			ws.logger.Warnw("WebSocket read error", "error", err)
			ws.triggerReconnect()
			return
		}

		msgStr := string(message)
		ws.logger.Debugw("WebSocket message", "msg", msgStr)

		// Handle PING from server → respond with PONG
		if msgStr == "PING" {
			ws.writeMu.Lock()
			writeErr := conn.WriteMessage(websocket.TextMessage, []byte("PONG"))
			ws.writeMu.Unlock()
			if writeErr != nil {
				ws.logger.Warnw("Failed to send PONG", "error", writeErr)
				ws.triggerReconnect()
				return
			}
			ws.logger.Debug("Received PING, sent PONG")
			continue
		}

		// Handle PONG response to our PING
		if msgStr == "PONG" {
			continue
		}

		ws.handleEvent(message)
	}
}

// handleEvent processes event(s) - data can be either a single object or an array
func (ws *WsClient) handleEvent(event json.RawMessage) {
	var events []json.RawMessage

	// Check if the event is an array or single object by inspecting the first byte
	if len(event) > 0 && event[0] == '[' {
		// It's an array
		if err := json.Unmarshal(event, &events); err != nil {
			ws.logger.Errorw("Failed to parse message array", "error", err, "raw", string(event))
			return
		}
	} else {
		// It's a single object, wrap it in an array for uniform processing
		events = []json.RawMessage{event}
	}

	// Process each event
	for _, msg := range events {
		// Parse event type from the message
		var eventMap map[string]interface{}
		if err := json.Unmarshal(msg, &eventMap); err != nil {
			ws.logger.Errorw("Failed to parse event object", "error", err, "raw", string(msg))
			continue
		}

		eventType, ok := eventMap["event_type"].(string)
		if !ok {
			ws.logger.Warnw("Event missing event_type field", "event", eventMap)
			continue
		}

		// Delegate to channel-specific handlers
		switch eventType {
		case "order", "trade":
			// User channel events
			ws.handleUserEvent(msg)
		default:
			// Other market events
			ws.handleMarketEvent(msg)
		}
	}
}

// triggerReconnect signals the reconnection handler to reconnect
func (ws *WsClient) triggerReconnect() {
	select {
	case ws.reconnect <- struct{}{}:
	default:
		// Already queued
	}
}

// handleReconnect handles automatic reconnection.
// Runs as a single goroutine for the lifetime of the WsClient.
func (ws *WsClient) handleReconnect(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.reconnect:
			ws.logger.Info("Reconnecting WebSocket...")

			// 1. Signal existing goroutines (readMessages, ping) to stop
			ws.connMu.Lock()
			select {
			case <-ws.done:
				// Already closed
			default:
				close(ws.done)
			}

			// 2. Close old connection
			if ws.conn != nil {
				ws.conn.Close()
				ws.conn = nil
			}
			ws.connMu.Unlock()

			// 3. Wait before reconnect (backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}

			// 4. Reconnect: dial + start new read/ping goroutines
			ws.connMu.Lock()
			err := ws.dialAndStart(ctx)
			ws.connMu.Unlock()

			if err != nil {
				ws.logger.Errorw("Reconnection failed", "error", err)
				// Retry reconnection after another delay
				ws.triggerReconnect()
				continue
			}

			ws.logger.Info("Reconnected successfully")

			// 5. Replay all stored subscriptions
			if err := ws.resubscribe(); err != nil {
				ws.logger.Errorw("Resubscription failed after reconnect", "error", err)
				ws.triggerReconnect()
			}
		}
	}
}

// storeSubscription saves a subscription for replay after reconnect
func (ws *WsClient) storeSubscription(key string, payload WsSubscription, handler WsHandler) {
	ws.storedSubsMu.Lock()
	defer ws.storedSubsMu.Unlock()

	// Replace existing subscription with same key
	for i, s := range ws.storedSubs {
		if s.Key == key {
			ws.storedSubs[i] = storedSubscription{Key: key, Payload: payload, Handler: handler}
			return
		}
	}

	ws.storedSubs = append(ws.storedSubs, storedSubscription{Key: key, Payload: payload, Handler: handler})
}

// removeSubscription removes a stored subscription by key
func (ws *WsClient) removeSubscription(key string) {
	ws.storedSubsMu.Lock()
	defer ws.storedSubsMu.Unlock()

	for i, s := range ws.storedSubs {
		if s.Key == key {
			ws.storedSubs = append(ws.storedSubs[:i], ws.storedSubs[i+1:]...)
			return
		}
	}
}

// resubscribe replays all stored subscriptions after reconnection
func (ws *WsClient) resubscribe() error {
	ws.storedSubsMu.Lock()
	subs := make([]storedSubscription, len(ws.storedSubs))
	copy(subs, ws.storedSubs)
	ws.storedSubsMu.Unlock()

	if len(subs) == 0 {
		ws.logger.Debug("No subscriptions to replay")
		return nil
	}

	for _, sub := range subs {
		ws.logger.Infow("Resubscribing", "key", sub.Key)

		// Re-register handler
		if sub.Handler != nil {
			ws.handlersMu.Lock()
			ws.handlers[sub.Key] = sub.Handler
			ws.handlersMu.Unlock()
		}

		// Re-send subscription message
		ws.connMu.RLock()
		conn := ws.conn
		ws.connMu.RUnlock()

		if conn == nil {
			return fmt.Errorf("connection lost during resubscription")
		}

		ws.writeMu.Lock()
		err := conn.WriteJSON(sub.Payload)
		ws.writeMu.Unlock()

		if err != nil {
			return fmt.Errorf("failed to resubscribe to %s: %w", sub.Key, err)
		}

		ws.logger.Infow("Resubscribed successfully", "key", sub.Key)
	}

	return nil
}
