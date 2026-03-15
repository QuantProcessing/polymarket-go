package clob

import (
	"context"
	"encoding/json"
	"fmt"
)

// SubscribeAccount subscribes to user channel for account-level events (orders, trades)
// This captures all activity for the authenticated account
func (ws *WsClient) SubscribeAccount(ctx context.Context, auth *WsAuthParams, handler WsHandler) error {
	ws.connMu.RLock()
	if ws.conn == nil {
		ws.connMu.RUnlock()
		return fmt.Errorf("not connected")
	}
	if ws.channelType != WsChannelUser {
		ws.connMu.RUnlock()
		return fmt.Errorf("wrong channel: expected 'user', got '%s'", ws.channelType)
	}
	ws.connMu.RUnlock()

	// Register handler for user channel
	// User channel doesn't subdivide by market - all account events go to same handler
	if handler != nil {
		ws.handlersMu.Lock()
		ws.handlers[WsChannelUser] = handler
		ws.handlersMu.Unlock()
	}

	// Send subscription message with authentication
	sub := WsSubscription{
		Type:    WsChannelUser,
		Auth:    auth,
		Markets: []string{}, // Empty = subscribe to all account activity
	}

	ws.writeMu.Lock()
	err := ws.conn.WriteJSON(sub)
	ws.writeMu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to subscribe to user channel: %w", err)
	}

	// Store subscription for replay after reconnect
	ws.storeSubscription(WsChannelUser, sub, handler)

	ws.logger.Infow("Subscribed to user channel", "key", WsChannelUser)
	return nil
}

// handleUserEvent processes user channel events (order, trade)
// This is called by handleEvent when event_type is "order" or "trade"
func (ws *WsClient) handleUserEvent(event json.RawMessage) {
	ws.handlersMu.RLock()
	defer ws.handlersMu.RUnlock()

	if handler, ok := ws.handlers[WsChannelUser]; ok {
		if err := handler(event); err != nil {
			ws.logger.Warnw("Handler error", "channel", WsChannelUser, "error", err, "event", event)
		}
		return
	}

	ws.logger.Debugw("No handler for user channel", "event", event)
}
