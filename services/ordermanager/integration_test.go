package ordermanager_test

import (
	"go.uber.org/zap"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantProcessing/polymarket-go"
	"github.com/QuantProcessing/polymarket-go/clients/clob"
	"github.com/QuantProcessing/polymarket-go/clients/gamma"
	"github.com/QuantProcessing/polymarket-go/services/ordermanager"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrderManager_Integration verifies OrderManager using real SDK components
func TestOrderManager_Integration(t *testing.T) {
	// 1. Setup
	_ = godotenv.Load("../../../../../.env")
	l := zap.NewNop().Sugar()
	ctx := context.Background()

	// 2. Initialize Root Client
	cfg := polymarket.Config{
		APIKey:        os.Getenv("CLOB_API_KEY"),
		APISecret:     os.Getenv("CLOB_SECRET"),
		APIPassphrase: os.Getenv("CLOB_PASSPHRASE"),
		PrivateKey:    os.Getenv("POLY_PRIVATE_KEY"),
		FunderAddress: os.Getenv("POLY_FUNDER_ADDR"), // Enable Proxy (User likely has funds here)
		ChainID:       137,                           // Polygon Mainnet
	}

	if cfg.PrivateKey == "" {
		t.Skip("Skipping OrderManager test (no private key)")
	}

	client, err := polymarket.NewClient(ctx, cfg, l)
	require.NoError(t, err)

	// 3a. Connect Realtime Service
	err = client.Realtime.ConnectAll()
	require.NoError(t, err)
	defer client.Realtime.CloseAll()

	// 3b. Subscribe to User Events (Required for OrderManager monitoring)
	// We make this optional for the test because local network/IP might block multiple WS connections
	// or authenticated WS. The OrderManager has polling fallback.
	err = client.Realtime.User.SubscribeUser(cfg.APIKey, cfg.APISecret, cfg.APIPassphrase, nil)
	if err != nil {
		t.Logf("Warning: Failed to subscribe to User WS (will use polling fallback): %v", err)
	} else {
		// Give it a moment to authenticate
		time.Sleep(2 * time.Second)
	}

	om := client.OrderManager
	require.NotNil(t, om, "OrderManager should be initialized")

	// 3. Find a Market to Trade
	// We need a valid TokenID. Let's search via Gamma for active markets with volume.
	active := true
	closed := false
	markets, err := client.Market.GetMarkets(ctx, gamma.MarketSearchParams{
		Limit:     5,
		Active:    &active,
		Closed:    &closed,
		Order:     "volume24hr",
		Ascending: func() *bool { b := false; return &b }(),
	})
	require.NoError(t, err)
	if len(markets) == 0 {
		t.Skip("No active markets found to test order placement")
	}

	// Find one with valid tokens
	var market gamma.GammaMarket
	var tokens []gamma.GammaToken
	found := false

	for _, m := range markets {
		ts := m.Tokens()
		if len(ts) >= 2 {
			market = m
			tokens = ts
			found = true
			break
		}
	}

	if !found {
		t.Skip("No markets with sufficient tokens found")
	}
	if len(tokens) < 2 {
		t.Skip("Market has insufficient tokens")
	}
	tokenID := tokens[0].TokenID // YES token usually

	t.Logf("Testing with Market: %s, TokenID: %s", market.Question, tokenID)

	// 4. Create Limit Order
	// Try with NegRisk=false first (standard)
	req := ordermanager.OrderRequest{
		TokenID:   tokenID,
		Price:     0.05,
		Size:      50.0, // 50 * 0.05 = $2.50 > $1 min
		Side:      clob.SideBuy,
		OrderType: "GTC",
		NegRisk:   false,
	}

	result, err := om.CreateOrder(ctx, req)
	if err != nil {
		t.Logf("CreateOrder (NegRisk=false) failed: %v. Retrying with NegRisk=true...", err)
		req.NegRisk = true
		result, err = om.CreateOrder(ctx, req)
		if err != nil {
			// Check if error is due to balance
			if strings.Contains(err.Error(), "not enough balance") || strings.Contains(err.Error(), "allowance") {
				t.Logf("CreateOrder failed due to insufficient funds/allowance (EXPECTED for empty test wallet): %v", err)
				t.Skip("Skipping execution verification due to insufficient funds")
				return
			}
			t.Fatalf("CreateOrder failed (both NegRisk modes): %v", err)
		}
	}

	assert.NotEmpty(t, result.OrderID)
	assert.Equal(t, ordermanager.StatusCreated, result.Status)
	assert.NotNil(t, result.Handle)

	t.Logf("Order Created: %s (NegRisk=%v)", result.OrderID, req.NegRisk)

	// 5. Watch for "created" state (immediate)
	select {
	case update := <-result.Handle.Updates:
		t.Logf("Received Initial Update: %s", update.Status)
	case <-time.After(5 * time.Second):
		t.Log("No immediate update on channel (expected if no WS event yet)")
	}

	// 6. Cancel Order
	time.Sleep(1 * time.Second) // Let it propagate active state
	err = om.CancelOrder(ctx, result.OrderID)
	assert.NoError(t, err)
	t.Log("Order Cancel request sent")

	// 7. Wait for Terminal State (Cancelled)
	// The Updates channel should be closed or emit StatusCancelled
	timeout := time.After(10 * time.Second)
	terminalReached := false

	for {
		select {
		case update, ok := <-result.Handle.Updates:
			if !ok {
				t.Log("Updates channel closed")
				terminalReached = true
				goto Done
			}
			t.Logf("Received Update: %s", update.Status)
			if update.Status == ordermanager.StatusCancelled {
				terminalReached = true
				// We expect the channel to be closed shortly after, or we can break here
			}
		case <-timeout:
			t.Fatal("Timeout waiting for order cancellation update")
		}
	}
Done:
	if !terminalReached {
		t.Fatal("Loop finished without reaching terminal state")
	}
	t.Log("Order correctly reached terminal state")

	// 7. Verify Cancellation update?
	// If WS is connected, we might see it.
	// We verify API call succeeded via `assert.NoError`.
}
