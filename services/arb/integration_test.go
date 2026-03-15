package arb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/QuantProcessing/polymarket-go"
	"github.com/QuantProcessing/polymarket-go/clients/gamma"
)

// TestArbitrageService_Integration validates the service lifecycle
// It connects to real Polygon/Polymarket components if env vars are present.
func TestArbitrageService_Integration(t *testing.T) {
	// 1. Setup Client — read from env
	privateKey := os.Getenv("POLY_PRIVATE_KEY")
	if privateKey == "" {
		t.Skip("Skipping integration test: POLY_PRIVATE_KEY not set")
	}

	logger := zap.NewNop().Sugar()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sdkConfig := polymarket.Config{
		APIKey:        os.Getenv("POLY_API_KEY"),
		APISecret:     os.Getenv("POLY_API_SECRET"),
		APIPassphrase: os.Getenv("POLY_PASSPHRASE"),
		PrivateKey:    privateKey,
		FunderAddress: os.Getenv("POLY_FUNDER_ADDRESS"),
		ChainID:       137, // Polygon Mainnet
	}

	pClient, err := polymarket.NewClient(ctx, sdkConfig, logger)
	require.NoError(t, err)

	// Connect Realtime Service
	err = pClient.Realtime.ConnectAll()
	require.NoError(t, err)

	// 2. Setup Arb Service
	arbCfg := DefaultConfig()
	arbCfg.AutoExecute = false // Safety
	arbCfg.EnableRebalancer = false

	arbService := NewArbitrageService(ctx, arbCfg, pClient.Trading, pClient.Ctf, pClient.Realtime.Market, logger)

	// 3. Start
	// Use a known active market
	market := MarketConfig{
		Name:        "Test Market",
		ConditionID: "0x...", // Need real ConditionID
		YESTokenID:  "0x...", // Need real Token IDs
		NOTokenID:   "0x...",
	}

	// Lookup a market via Gamma
	clobMarkets, err := pClient.Market.Gamma.GetMarkets(context.Background(), gamma.MarketSearchParams{Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, clobMarkets)

	target := clobMarkets[0]
	if len(target.Tokens()) < 2 {
		t.Skip("Market doesn't have 2 tokens")
	}

	market.ConditionID = target.ConditionID
	market.YESTokenID = target.Tokens()[0].TokenID
	market.NOTokenID = target.Tokens()[1].TokenID
	market.Name = target.Question

	err = arbService.Start(market)
	require.NoError(t, err)

	// 4. Wait for RTDS messages
	time.Sleep(5 * time.Second)

	// 5. Stop
	arbService.Stop()

	// Assert stats (might be 0 opportunities, which is fine)
	stats := arbService.stats
	t.Logf("Stats: %+v", stats)

	assert.False(t, arbService.isRunning)
}
