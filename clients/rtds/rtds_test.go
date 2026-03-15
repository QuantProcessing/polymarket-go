package rtds

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTDSClient_Integration_CryptoPrices(t *testing.T) {
	// Skip in short mode as this connects to real WS
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Initialize Client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := RTDSConfig{
		URL:           RTDSLiveDataURL,
		AutoReconnect: true,
	}
	rtds := NewRTDSClient(ctx, config)
	
	// 2. Connect
	err := rtds.Connect()
	require.NoError(t, err)
	defer rtds.Close()

	// 3. Subscribe to Chainlink Prices
	// We'll use a channel to capture the update
	priceUpdateCh := make(chan *CryptoPricePayload, 1)
	
	// Create a filter for BTC/USD
	err = rtds.SubscribeCryptoPricesChainlink([]string{"btc/usd"})
	require.NoError(t, err)

	// 4. Wait for updates
	// Since the client uses a generic channel, we need to parse messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-rtds.Messages():
				var update CryptoPriceUpdate
				if err := json.Unmarshal(msg, &update); err == nil {
					if update.Topic == "crypto_prices_chainlink" && update.Payload.Symbol == "btc/usd" {
						priceUpdateCh <- &update.Payload
						return
					}
				}
			}
		}
	}()

	// 5. Assert
	select {
	case price := <-priceUpdateCh:
		t.Logf("Received Price Update: %+v", price)
		assert.Equal(t, "btc/usd", price.Symbol)
		assert.Greater(t, price.Value, 0.0)
		assert.Greater(t, price.Timestamp, int64(0))
	case <-time.After(15 * time.Second):
		t.Fatal("Timeout waiting for crypto price update")
	}
}

func TestRTDSClient_Integration_Market(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use Market URL
	config := RTDSConfig{
		URL:           RTDSMarketURL,
		AutoReconnect: true,
	}
	rtds := NewRTDSClient(ctx, config)

	err := rtds.Connect()
	require.NoError(t, err)
	defer rtds.Close()
	
	// Subscribe to a populated market to ensure traffic
	// Note: Asset IDs change, so we might need a known active market or just check connection
	// For now, we just verify we can send the subscription command without error
	err = rtds.SubscribeMarket([]string{"21742633143463906290569050155826241533067272736897614950488156847949938836455"})
	require.NoError(t, err)
}
