package ctf

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantProcessing/polymarket-go/clients/gamma"
)

func TestCTFClient_Constants(t *testing.T) {
	assert.Equal(t, "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045", CTFContractAddress)
	assert.Equal(t, "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", USDCContractAddress)
}

func TestCTFClient_Integration_Workflow(t *testing.T) {
	// This test requires a real private key and RPC URL to run actual transactions.
	// It uses GammaClient to fetch real market data.
	// We load .env if present
	_ = godotenv.Load("../../../../../.env")

	privateKey := os.Getenv("POLY_PRIVATE_KEY")
	rpcURL := os.Getenv("POLY_RPC_URL")
	funderAddr := os.Getenv("POLY_FUNDER_ADDR")
	if privateKey == "" || rpcURL == "" || funderAddr == "" {
		t.Skip("POLY_PRIVATE_KEY, POLY_RPC_URL, or POLY_FUNDER_ADDR not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Setup GammaClient to get real market data
	gammaClient := gamma.NewGammaClient(http.DefaultClient)
	eventSlug := "us-iran-nuclear-deal-before-2027"
	event, err := gammaClient.GetEventBySlug(ctx, eventSlug)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.NotEmpty(t, event.Markets, "Event should have markets")
	
	market := event.Markets[0]
	t.Logf("Using market: %s (ConditionID: %s)", market.Question, market.ConditionID)

	// 2. Setup CTFClient
	ctfClient, err := NewCTFClient(CTFClientConfig{
		PrivateKeyHex: privateKey,
		RPCURL:        rpcURL,
		ChainID:       137, // Polygon Mainnet
		FunderAddress: funderAddr, // Use proxy address
	})
	require.NoError(t, err)

	// 3. Check Readiness
	t.Logf("Wallet Address (EOA): %s", ctfClient.GetAddress())
	t.Logf("Funder Address (Proxy): %s", ctfClient.GetFunderAddress())
	
	maticBal, _ := ctfClient.client.BalanceAt(ctx, ctfClient.address, nil)
	t.Logf("EOA MATIC Balance: %s", decimal.NewFromBigInt(maticBal, -18).String())
	
	nativeBal, _ := ctfClient.GetNativeUsdcBalance(ctx)
	t.Logf("Funder Native USDC Balance: %s", nativeBal.String())
	
	usdcEBal, _ := ctfClient.GetUsdcBalance(ctx)
	t.Logf("Funder USDC.e Balance: %s", usdcEBal.String())

	ready, msg, err := ctfClient.CheckReadyForCTF(ctx, "0.1")
	require.NoError(t, err)
	t.Logf("Ready status: %v. Msg: %s", ready, msg)
	if !ready {
		t.Skipf("Not ready for CTF ops: %s", msg)
	}

	// 4. Test GetPositionBalance (Read-only)
	bal, err := ctfClient.GetPositionBalance(ctx, market.ConditionID)
	require.NoError(t, err)
	t.Logf("Balances: YES=%s, NO=%s", bal.YesBalance, bal.NoBalance)

	// 5. Test Split (Write) - careful with real funds!
	// Only run if specific flag is set to avoid accidental spending
	// CTFClient now supports both EOA and Proxy Wallet modes automatically
	if os.Getenv("ENABLE_SPEND_TESTS") == "true" {
		amount := "0.1" // Small amount
		if ctfClient.GetAddress() != ctfClient.GetFunderAddress() {
			t.Log("Testing Proxy Wallet execution via ProxyWalletFactory...")
		} else {
			t.Log("Testing direct EOA execution...")
		}
		t.Logf("Attempting to split %s USDC...", amount)
		txHash, err := ctfClient.Split(ctx, market.ConditionID, amount)
		require.NoError(t, err)
		t.Logf("Split Tx: %s", txHash)

		// Wait for mining? In real test we might want to wait.
		time.Sleep(5 * time.Second)

		// 6. Test Merge (Write)
		t.Logf("Attempting to merge %s sets...", amount)
		txHashMerge, err := ctfClient.Merge(ctx, market.ConditionID, amount)
		require.NoError(t, err)
		t.Logf("Merge Tx: %s", txHashMerge)
	} else {
		t.Log("Skipping Split/Merge/Redeem (ENABLE_SPEND_TESTS not set)")
	}

	// 7. Test GetPositionBalanceByTokenIds if we have token IDs
	// Not all Gamma markets have clobTokenIds directly available in the structure I defined earlier?
	// The GammaMarket struct has `id` and `conditionId`.
	// To test `RedeemByTokenIds` or `GetPositionBalanceByTokenIds`, we'd need valid Token IDs.
	// For now, we verified the methods compile.
}
