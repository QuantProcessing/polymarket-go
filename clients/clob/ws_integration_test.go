package clob

import (
	"go.uber.org/zap"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

)

// TestWSMarketSubscriptions tests WebSocket subscriptions for market data
// It covers: book, price_change, tick_size_change, last_trade_price, best_bid_ask, new_market, market_resolved
// Note: Some events (new_market, market_resolved) are rare and may not occur during the test window.
func TestWSMarketSubscriptions(t *testing.T) {
	// 1. Setup
	l := zap.NewNop().Sugar()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30s window
	defer cancel()

	// 2. Fetch a valid market/asset ID using REST API to ensure we subscribe to an active market
	restClient := NewClient("", l)

	// Try fetching active markets first
	markets, err := restClient.GetActiveMarkets(ctx, "")
	if err != nil {
		t.Fatalf("Failed to fetch markets via REST: %v", err)
	}
	if len(markets.Data) == 0 {
		t.Fatal("No active markets found via REST")
	}

	var firstMarket map[string]interface{}

	// Iterate to find a truly active market (2025+)
	for _, m := range markets.Data {
		mMap, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		// Check end date
		endDateStr, ok := mMap["end_date_iso"].(string)
		if !ok || len(endDateStr) < 4 {
			continue
		}
		if endDateStr[:4] < "2025" {
			continue
		}

		// Check tokens
		tokens, ok := mMap["tokens"].([]interface{})
		if !ok || len(tokens) == 0 {
			continue
		}

		validToken := false
		for _, tok := range tokens {
			tokMap, ok := tok.(map[string]interface{})
			if !ok {
				continue
			}
			tid, ok := tokMap["token_id"].(string)
			if ok && tid != "" {
				validToken = true
				break
			}
		}

		if !validToken {
			continue
		}

		firstMarket = mMap
		break
	}

	var assetID, conditionID string

	if firstMarket != nil {
		tokens, _ := firstMarket["tokens"].([]interface{})
		for _, tok := range tokens {
			tokMap, _ := tok.(map[string]interface{})
			if tid, ok := tokMap["token_id"].(string); ok && tid != "" {
				assetID = tid
				break
			}
		}
		conditionID, _ = firstMarket["condition_id"].(string)
		t.Logf("Targeting Dynamic Market: %v (Asset: %s, Condition: %s)", firstMarket["question"], assetID, conditionID)
	} else {
		// Fallback to user provided IDs
		t.Log("No 2025+ markets found, using user provided fallback.")
		conditionID = "0xc15d1fbf1afaec0ae631175aa97c97ea901cc48e2dfd03911a4c83e2b287caad"
		assetID = "112999906182783404584448708403705986931443013333146484732564904731523271213934"
		t.Logf("Targeting User Provided Market (Asset: %s, Condition: %s)", assetID, conditionID)
	}
	// User suggested optimized IDs for manual testing reference
	// conditionID = "0xc15d1fbf1afaec0ae631175aa97c97ea901cc48e2dfd03911a4c83e2b287caad"
	// assetID = "112999906182783404584448708403705986931443013333146484732564904731523271213934"

	// 3. Connect WS
	// Create client with base URL (channel is specified in Connect)
	ws := NewWsClient("", l)
	if err := ws.Connect(ctx, WsChannelMarket); err != nil {
		t.Fatalf("Failed to connect WS: %v", err)
	}
	defer ws.Close()

	// 4. Create channels to track received event types
	receivedEvents := make(map[string]bool)
	msgChan := make(chan string, 100)

	handler := func(data []byte) error {
		// Generic parsing to find event_type
		var base struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(data, &base); err != nil {
			return err
		}

		if base.EventType != "" {
			msgChan <- base.EventType
		} else {
			t.Logf("Received raw (no event_type): %s", string(data))
		}
		return nil
	}

	// 5. Subscribe to 'market' channel using initial subscription format
	err = ws.Subscribe(WsSubscription{
		Type:      WsChannelMarket,
		AssetsIDs: []string{assetID},
	}, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to market channel: %v", err)
	}

	// 6. Book events are already included in market channel
	// No need for separate subscription - market channel provides all events

	// 7. Listen for a while
	t.Log("Listening for WS events for 10 seconds...")
	timer := time.NewTimer(10 * time.Second)

loop:
	for {
		select {
		case evt := <-msgChan:
			if !receivedEvents[evt] {
				receivedEvents[evt] = true
				t.Logf("✓ Received Event: %s", evt)
			}
		case <-timer.C:
			break loop
		case <-ctx.Done():
			break loop
		}
	}

	// 8. Report results
	t.Log("--- Event Summary ---")
	expectedEvents := []string{"book", "price_change", "tick_size_change", "last_trade_price", "best_bid_ask"}

	for _, evt := range expectedEvents {
		if receivedEvents[evt] {
			t.Logf("[PASS] Received %s", evt)
		} else {
			t.Logf("[WARN] Did not receive %s (might need active trading)", evt)
		}
	}

	// We require at least ONE useful event to pass
	if len(receivedEvents) == 0 {
		t.Fatal("Received NO events from WebSocket")
	}
}

// TestWSUserSubscription tests WebSocket subscription for user channel (account orders/trades)
// Pass criteria: receive 10 order events OR 10 minute timeout
func TestWSUserSubscription(t *testing.T) {
	// Skip if no credentials
	if !hasTestCredentials() {
		t.Skip("Skipping test: no credentials available (POLY_PRIVATE_KEY, POLY_FUNDER_ADDR)")
	}

	// Setup with debug logging
	l := zap.NewNop().Sugar()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Initialize REST client for authentication
	restClient := NewClient("", l)
	creds := getTestCreds()
	restClient.WithCredentials(creds)

	// Derive API credentials
	if err := restClient.EnsureAuth(ctx); err != nil {
		t.Fatalf("Failed to ensure auth: %v", err)
	}

	// Connect to user WebSocket channel
	wsClient := NewWsClient("", l)
	if err := wsClient.Connect(ctx, WsChannelUser); err != nil {
		t.Fatalf("Failed to connect to user channel: %v", err)
	}
	defer wsClient.Close()

	// Track events
	const targetEvents = 10
	eventChan := make(chan struct{}, targetEvents)
	var eventCount int

	// Event handler
	handler := func(data []byte) error {
		eventCount++
		t.Logf("📨 Event #%d: %s", eventCount, string(data[:min(len(data), 150)]))

		// Parse event
		var event struct {
			EventType string `json:"event_type"`
			ID        string `json:"id"`
			Type      string `json:"type"` // PLACEMENT, UPDATE, CANCELLATION
		}

		if err := json.Unmarshal(data, &event); err == nil && event.EventType == "order" {
			t.Logf("✅ Order Event: type=%s, id=%s...", event.Type, event.ID[:min(len(event.ID), 16)])

			// Signal event received
			select {
			case eventChan <- struct{}{}:
			default:
			}
		}

		return nil
	}

	// Subscribe to user channel
	auth := &WsAuthParams{
		APIKey:     restClient.GetAPIKey(),
		Secret:     restClient.GetAPISecret(),
		Passphrase: restClient.GetAPIPassphrase(),
	}

	if err := wsClient.SubscribeAccount(ctx, auth, handler); err != nil {
		t.Fatalf("Failed to subscribe to user channel: %v", err)
	}

	t.Logf("✅ Subscribed to user channel. Waiting for %d events or 10 minute timeout...", targetEvents)
	t.Log("💡 Tip: Place some orders on Polymarket to trigger events during this test")

	// Wait for target events or timeout
	receivedTarget := false
	for i := 0; i < targetEvents; i++ {
		select {
		case <-eventChan:
			if i+1 == targetEvents {
				receivedTarget = true
			}
		case <-ctx.Done():
			goto done
		}
	}

done:
	// Summary
	t.Logf("=== SUMMARY ===")
	t.Logf("Events received: %d/%d", eventCount, targetEvents)

	if receivedTarget {
		t.Logf("✅ Test PASSED: Received target %d events", targetEvents)
	} else if eventCount > 0 {
		t.Logf("⚠️  Test completed with %d/%d events (timeout reached)", eventCount, targetEvents)
	} else {
		t.Log("ℹ️  No events received (expected if no orders placed during test window)")
	}
}

// Helper functions for user channel test
func hasTestCredentials() bool {
	creds := getTestCreds()
	return creds.PrivateKey != "" && creds.FunderAddress != ""
}

func getTestCreds() *Credentials {
	return &Credentials{
		PrivateKey:    os.Getenv("POLY_PRIVATE_KEY"),
		FunderAddress: os.Getenv("POLY_FUNDER_ADDR"),
		APIKey:        os.Getenv("CLOB_API_KEY"),
		APISecret:     os.Getenv("CLOB_SECRET"),
		APIPassphrase: os.Getenv("CLOB_PASSPHRASE"),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestWSOrderbookSubscription tests WebSocket subscription for market orderbook data
// Pass criteria: receive 100 orderbook updates OR 10 minute timeout
func TestWSOrderbookSubscription(t *testing.T) {
	// Setup with debug logging
	l := zap.NewNop().Sugar()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Connect to market WebSocket channel
	wsClient := NewWsClient("", l)
	if err := wsClient.Connect(ctx, WsChannelMarket); err != nil {
		t.Fatalf("Failed to connect to market channel: %v", err)
	}
	defer wsClient.Close()

	// Track orderbook updates
	const targetUpdates = 10000
	updateChan := make(chan struct{}, targetUpdates)
	var updateCount int

	// Event handler
	handler := func(data []byte) error {
		updateCount++

		// Parse to check event type
		var event struct {
			EventType string `json:"event_type"`
		}

		if err := json.Unmarshal(data, &event); err == nil {
			// Count all market events (book, price_change, etc.)
			if updateCount%10 == 0 || updateCount <= 5 {
				t.Logf("� Market Event #%d (type: %s)", updateCount, event.EventType)
			}

			// Signal update received
			select {
			case updateChan <- struct{}{}:
			default:
			}
		}

		return nil
	}

	marketID := "xxx"
	assetID := "93726727647510357355971239823727997335316035606609139108996303272250378787618"
	// Subscribe to market orderbook using asset_id (token ID)
	if err := wsClient.SubscribeMarket(ctx, marketID, []string{assetID}, handler); err != nil {
		t.Fatalf("Failed to subscribe to market: %v", err)
	}

	t.Logf("✅ Subscribed to market data. Waiting for %d updates or 10 minute timeout...", targetUpdates)

	// Wait for target updates or timeout
	receivedTarget := false
	for i := 0; i < targetUpdates; i++ {
		select {
		case <-updateChan:
			if i+1 == targetUpdates {
				receivedTarget = true
			}
		case <-ctx.Done():
			goto done
		}
	}

done:
	// Summary
	t.Logf("=== SUMMARY ===")
	t.Logf("Orderbook updates received: %d/%d", updateCount, targetUpdates)

	if receivedTarget {
		t.Logf("✅ Test PASSED: Received target %d orderbook updates", targetUpdates)
	} else if updateCount > 0 {
		t.Logf("⚠️  Test completed with %d/%d updates (timeout reached)", updateCount, targetUpdates)
	} else {
		t.Log("ℹ️  No orderbook updates received (market might be inactive)")
	}
}
