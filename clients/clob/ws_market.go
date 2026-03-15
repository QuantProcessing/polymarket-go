package clob

import (
	"context"
	"encoding/json"
	"fmt"
)

// SubscribeMarket subscribes to market channel for market data (orderbook, prices, trades)
// assetIDs: list of asset token IDs to subscribe to
func (ws *WsClient) SubscribeMarket(ctx context.Context, marketID string, assetIDs []string, handler WsHandler) error {
	ws.connMu.RLock()
	if ws.conn == nil {
		ws.connMu.RUnlock()
		return fmt.Errorf("not connected")
	}
	if ws.channelType != WsChannelMarket {
		ws.connMu.RUnlock()
		return fmt.Errorf("wrong channel: expected 'market', got '%s'", ws.channelType)
	}
	ws.connMu.RUnlock()

	// Register handlers for each asset using assetID as key
	key := fmt.Sprintf("market:%s", marketID)
	if handler != nil {
		ws.handlersMu.Lock()
		ws.handlers[key] = handler
		ws.handlersMu.Unlock()
	}

	// Send subscription message
	sub := WsSubscription{
		Operation:            "subscribe",
		AssetsIDs:            assetIDs,
		CustomFeatureEnabled: true,
	}

	ws.writeMu.Lock()
	err := ws.conn.WriteJSON(sub)
	ws.writeMu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to subscribe to market channel: %w", err)
	}

	// Store subscription for replay after reconnect
	ws.storeSubscription(key, sub, handler)

	ws.logger.Infow("Subscribed to market channel", "assets", assetIDs)
	return nil
}

// UnsubscribeMarket unsubscribes from specific assets
func (ws *WsClient) UnsubscribeMarket(assetIDs []string) error {
	ws.connMu.RLock()
	if ws.conn == nil {
		ws.connMu.RUnlock()
		return fmt.Errorf("not connected")
	}
	ws.connMu.RUnlock()

	// Build unsubscribe message
	unsub := WsSubscription{
		Operation: "unsubscribe",
		AssetsIDs: assetIDs,
	}

	ws.writeMu.Lock()
	err := ws.conn.WriteJSON(unsub)
	ws.writeMu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	// Remove handlers using assetID as key
	ws.handlersMu.Lock()
	for _, assetID := range assetIDs {
		delete(ws.handlers, assetID)
	}
	ws.handlersMu.Unlock()

	ws.logger.Infow("Unsubscribed from market assets", "assets", assetIDs)
	return nil
}

// handleMarketEvent processes market channel events (book, price changes, etc.)
func (ws *WsClient) handleMarketEvent(event json.RawMessage) {
	ws.handlersMu.RLock()
	defer ws.handlersMu.RUnlock()

	var msgStruct struct {
		EventType string `json:"event_type"`
		Market    string `json:"market"`
	}

	if err := json.Unmarshal(event, &msgStruct); err != nil {
		ws.logger.Errorw("Failed to parse message", "error", err, "event", event)
		return
	}

	key := fmt.Sprintf("market:%s", msgStruct.Market)
	if handler, ok := ws.handlers[key]; ok {
		if err := handler(event); err != nil {
			ws.logger.Warnw("Handler error", "channel", WsChannelMarket, "error", err, "event", event)
		}
		return
	}

	ws.logger.Debugw("No handler for market channel", "event", event)
}
