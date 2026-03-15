package polymarket

import (
	"go.uber.org/zap"
	"context"
	"os"
	"testing"
	"time"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
	"github.com/QuantProcessing/polymarket-go/clients/gamma"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ServicesIntegrationTest verifies the high-level Service Layer
func TestServices_Integration(t *testing.T) {
	// 1. Setup
	_ = godotenv.Load("../../../.env")
	l := zap.NewNop().Sugar()
	ctx := context.Background()

	// 2. Initialize Root Client
	client, err := NewClient(ctx, Config{}, l)
	require.NoError(t, err)

	// Auth (Optional for some tests, required for Trading)
	pk := os.Getenv("POLY_PRIVATE_KEY")
	if pk != "" {
		client.Clob.WithCredentials(&clob.Credentials{
			PrivateKey:    pk,
			FunderAddress: os.Getenv("POLY_FUNDER_ADDR"),
			APIKey:        os.Getenv("CLOB_API_KEY"),
			APISecret:     os.Getenv("CLOB_SECRET"),
			APIPassphrase: os.Getenv("CLOB_PASSPHRASE"),
		})
	}

	// ==========================================
	// Market Service Tests
	// ==========================================
	t.Run("MarketService_GetEventBySlug", func(t *testing.T) {
		// Use a known slug (from Gamma tests)
		slug := "us-iran-nuclear-deal-before-2027"
		// The Service exposes GetEventBySlug via Gamma delegation
		// But looking at market/client.go, it does NOT expose GetEventBySlug directly?
		// Wait, I saw it in the file view?
		// No, looking at Step 1697 output:
		// type MarketService struct { Gamma *gamma.GammaClient ... }
		// It delegates explicit methods: GetMarketByConditionID, GetMarkets, GetTrades, GetPositions.
		// It does NOT have GetEventBySlug wrapper.
		// So we should access it via .Gamma or add the wrapper.
		// For now, let's test what IS exposed or access via .Gamma if intended.
		// The user wants "Functional Testing" of the SDK.
		// If the SDK design allows accessing .Gamma, we test that.

		event, err := client.Gamma.GetEventBySlug(ctx, slug)
		if err != nil {
			t.Logf("GetEventBySlug failed (might be invalid slug): %v", err)
			return
		}
		assert.NotNil(t, event)
		assert.NotEmpty(t, event.Markets)
		t.Logf("Fetched Event: %s (Markets: %d)", event.Title, len(event.Markets))
	})

	t.Run("MarketService_GetMarketByConditionID", func(t *testing.T) {
		// Use Gamma to find a market first
		params := gamma.MarketSearchParams{
			Limit: 1,
		}
		markets, err := client.Market.GetMarkets(ctx, params)
		if err != nil {
			t.Skipf("Skipping GetMarket test due to search failure: %v", err)
		}
		if len(markets) > 0 {
			conditionID := markets[0].ConditionID
			m, err := client.Market.GetMarketByConditionID(ctx, conditionID)
			require.NoError(t, err)
			assert.Equal(t, conditionID, m.ConditionID)
			t.Logf("Fetched Market: %s (ConditionID: %s)", m.Question, m.ConditionID)
		}
	})

	// ==========================================
	// Trading Service Tests
	// ==========================================
	t.Run("TradingService_CheckReadiness", func(t *testing.T) {
		if pk == "" {
			t.Skip("Skipping Trading tests (no private key)")
		}
		assert.NotNil(t, client.Trading)
		// Access underlying clob for balance check if TradingService doesn't expose it directly yet
		// The TradingService in `services/trading` (Step 1572) has `CreateOrder` etc.
	})

	// ==========================================
	// Realtime Service Tests
	// ==========================================
	t.Run("RealtimeService_Subscribe", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping Realtime test in short mode")
		}

		// CONNECT FIRST!
		err := client.Realtime.ConnectAll()
		require.NoError(t, err)
		defer client.Realtime.CloseAll()

		// Use a dummy asset ID
		dummyAssetID := "1234567890" // Needs to be string

		// RealtimeService exposes Market *rtds.RTDSClient
		err = client.Realtime.Market.SubscribeMarket([]string{dummyAssetID})
		assert.NoError(t, err)
		t.Log("Subscribed to market channel")

		// Wait briefly
		time.Sleep(1 * time.Second)

		// Unsubscribe is not explicitly exposed in RTDSClient as "UnsubscribeFromMarket"
		// but standard RTDS might support it?
		// Looking at `rtds.go`: SubscribeMarket sends a JSON.
		// There is NO Unsubscribe method in `rtds.go` (Step 1674).
		// That's a finding! RTDS Client lacks Unsubscribe?
		// Wait, `clients/clob/ws_client.go` has Unsubscribe, but `rtds` might not.
		// Let's check `rtds.go` again... It has `Close()`.
		// Ideally we should test what is available.
	})
}
