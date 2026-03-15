package polymarket

import (
	"go.uber.org/zap"
	"context"
	"os"
	"testing"
	"time"

	"github.com/QuantProcessing/polymarket-go/clients/clob"

	"github.com/joho/godotenv"
)

// Integration test suite for all public SDK methods
// Requires .env file with valid credentials:
// - POLY_PRIVATE_KEY
// - POLY_FUNDER_ADDR
// - CLOB_API_KEY (optional)
// - CLOB_SECRET (optional)
// - CLOB_PASSPHRASE (optional)

var (
	testClient *Client
	testCreds  *clob.Credentials
)

func init() {
	// Load .env file
	_ = godotenv.Load("../../../.env")


	// Setup test client
	ctx := context.Background()
	cfg := Config{}
	l := zap.NewNop().Sugar()
	var err error
	testClient, err = NewClient(ctx, cfg, l)
	if err != nil {
		panic(err)
	}

	// Load credentials from environment
	privateKey := os.Getenv("POLY_PRIVATE_KEY")
	funderAddr := os.Getenv("POLY_FUNDER_ADDR")
	apiKey := os.Getenv("CLOB_API_KEY")
	secret := os.Getenv("CLOB_SECRET")
	passphrase := os.Getenv("CLOB_PASSPHRASE")

	if privateKey != "" && funderAddr != "" {
		testCreds = &clob.Credentials{
			PrivateKey:    privateKey,
			FunderAddress: funderAddr,
			APIKey:        apiKey,
			APISecret:     secret,
			APIPassphrase: passphrase,
		}
		testClient.Clob.WithCredentials(testCreds)
	}
}

// ========================================
// Market Data Tests (market.go)
// ========================================

func TestMarket_GetServerTime(t *testing.T) {
	ctx := context.Background()

	ts, err := testClient.Clob.GetServerTime(ctx)
	if err != nil {
		t.Fatalf("GetServerTime failed: %v", err)
	}

	if ts <= 0 {
		t.Fatalf("Expected positive timestamp, got: %d", ts)
	}

	// Verify timestamp is within reasonable range (past hour to future hour)
	now := time.Now().Unix()
	if ts < now-3600 || ts > now+3600 {
		t.Errorf("Server time %d seems incorrect (now: %d)", ts, now)
	}

	t.Logf("✓ GetServerTime: %d", ts)
}

func TestMarket_GetMarkets(t *testing.T) {
	ctx := context.Background()

	result, err := testClient.Clob.GetMarkets(ctx, "")
	if err != nil {
		t.Fatalf("GetMarkets failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	t.Logf("✓ GetMarkets: next_cursor=%s, data_count=%d", result.NextCursor, len(result.Data))
}

func TestMarket_GetSimplifiedMarkets(t *testing.T) {
	ctx := context.Background()

	result, err := testClient.Clob.GetSimplifiedMarkets(ctx, "")
	if err != nil {
		t.Fatalf("GetSimplifiedMarkets failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	t.Logf("✓ GetSimplifiedMarkets: next_cursor=%s, data_count=%d", result.NextCursor, len(result.Data))
}

func TestMarket_GetSamplingMarkets(t *testing.T) {
	ctx := context.Background()

	result, err := testClient.Clob.GetSamplingMarkets(ctx, "")
	if err != nil {
		t.Fatalf("GetSamplingMarkets failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	t.Logf("✓ GetSamplingMarkets: next_cursor=%s, data_count=%d", result.NextCursor, len(result.Data))
}

// Note: The following tests require valid market/token IDs
// We'll use a well-known condition ID for testing

const testConditionID = "0x139bc939a90e0414f2446ef09b30e3c5a14cc0fa8496c11bb6822954ad674a17"        // will-the-us-invade-venezuela-in-2025
const testTokenID = "54544668003738758433824606108816646836259157102970789856801480211062827461038" // will-the-us-invade-venezuela-in-2025 YES token

func TestMarket_GetMarket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	market, err := testClient.Clob.GetMarket(ctx, testConditionID)
	if err != nil {
		t.Fatalf("GetMarket failed: %v", err)
	}

	if market == nil {
		t.Fatal("Expected non-nil market")
	}

	if market.ConditionID != testConditionID {
		t.Errorf("Expected condition_id %s, got %s", testConditionID, market.ConditionID)
	}

	t.Logf("✓ GetMarket: %s - %s", market.ConditionID, market.Question)
}

func TestMarket_GetOrderBook(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	book, err := testClient.Clob.GetOrderBook(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetOrderBook failed: %v", err)
	}

	if book == nil {
		t.Fatal("Expected non-nil orderbook")
	}

	t.Logf("✓ GetOrderBook: asset=%s, bids=%d, asks=%d", book.AssetID, len(book.Bids), len(book.Asks))
}

func TestMarket_GetMidpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	mid, err := testClient.Clob.GetMidpoint(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetMidpoint failed: %v", err)
	}

	if mid <= 0 || mid >= 1 {
		t.Errorf("Midpoint %v out of expected range (0, 1)", mid)
	}

	t.Logf("✓ GetMidpoint: %v", mid)
}

func TestMarket_GetPrice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	price, err := testClient.Clob.GetPrice(ctx, testTokenID, clob.SideBuy)
	if err != nil {
		t.Fatalf("GetPrice failed: %v", err)
	}

	if price < 0 || price > 1 {
		t.Errorf("Price %v out of expected range [0, 1]", price)
	}

	t.Logf("✓ GetPrice (BUY): %v", price)
}

func TestMarket_GetSpread(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	spread, err := testClient.Clob.GetSpread(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetSpread failed: %v", err)
	}

	if spread < 0 {
		t.Errorf("Spread %v should be non-negative", spread)
	}

	t.Logf("✓ GetSpread: %v", spread)
}

func TestMarket_GetLastTradePrice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	price, err := testClient.Clob.GetLastTradePrice(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetLastTradePrice failed: %v", err)
	}

	if price < 0 || price > 1 {
		t.Errorf("Last trade price %v out of expected range [0, 1]", price)
	}

	t.Logf("✓ GetLastTradePrice: %v", price)
}

func TestMarket_GetTrades(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Note: This endpoint requires L2 API credentials which are automatically derived
	trades, err := testClient.Clob.GetTrades(ctx, clob.TradeParams{
		Asset: testTokenID,
	})
	if err != nil {
		t.Fatalf("GetTrades failed: %v", err)
	}

	if trades == nil {
		t.Fatal("Expected non-nil trades")
	}

	t.Logf("✓ GetTrades: %d trades", len(trades))
}

func TestMarket_GetPricesHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	now := time.Now()
	startTS := now.Add(-24 * time.Hour).Unix()

	prices, err := testClient.Clob.GetPricesHistory(ctx, clob.PriceHistoryFilterParams{
		Market:  testConditionID,
		StartTS: startTS,
		EndTS:   now.Unix(),
	})
	if err != nil {
		t.Fatalf("GetPricesHistory failed: %v", err)
	}

	if prices == nil {
		t.Fatal("Expected non-nil prices")
	}

	t.Logf("✓ GetPricesHistory: %d data points", len(prices))
}

// ========================================
// Order Management Tests (order.go)
// ========================================

func TestOrder_GetTickSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	tickSize, err := testClient.Clob.GetTickSize(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetTickSize failed: %v", err)
	}

	if tickSize == "" {
		t.Fatal("Expected non-empty tick size")
	}

	// Test caching - second call should be instant
	tickSize2, err := testClient.Clob.GetTickSize(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetTickSize (cached) failed: %v", err)
	}

	if tickSize != tickSize2 {
		t.Errorf("Cached tick size mismatch: %s != %s", tickSize, tickSize2)
	}

	t.Logf("✓ GetTickSize: %s (cached: %s)", tickSize, tickSize2)
}

func TestOrder_GetNegRisk(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	negRisk, err := testClient.Clob.GetNegRisk(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetNegRisk failed: %v", err)
	}

	// Test caching
	negRisk2, err := testClient.Clob.GetNegRisk(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetNegRisk (cached) failed: %v", err)
	}

	if negRisk != negRisk2 {
		t.Errorf("Cached neg risk mismatch: %v != %v", negRisk, negRisk2)
	}

	t.Logf("✓ GetNegRisk: %v (cached: %v)", negRisk, negRisk2)
}

func TestOrder_GetFeeRateBps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	feeRate, err := testClient.Clob.GetFeeRateBps(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetFeeRateBps failed: %v", err)
	}

	if feeRate < 0 {
		t.Errorf("Fee rate should be non-negative, got: %d", feeRate)
	}

	// Test caching
	feeRate2, err := testClient.Clob.GetFeeRateBps(ctx, testTokenID)
	if err != nil {
		t.Fatalf("GetFeeRateBps (cached) failed: %v", err)
	}

	if feeRate != feeRate2 {
		t.Errorf("Cached fee rate mismatch: %d != %d", feeRate, feeRate2)
	}

	t.Logf("✓ GetFeeRateBps: %d bps (cached: %d)", feeRate, feeRate2)
}

func TestOrder_GetOpenOrders(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	ctx := context.Background()

	orders, err := testClient.Clob.GetOpenOrders(ctx, clob.OpenOrderParams{})
	if err != nil {
		t.Fatalf("GetOpenOrders failed: %v", err)
	}

	if orders == nil {
		t.Fatal("Expected non-nil orders response")
	}

	t.Logf("✓ GetOpenOrders: %d open orders, next_cursor=%s", len(orders.Data), orders.NextCursor)
	for _, order := range orders.Data {
		t.Logf("Order: %v", order)
	}
}

func TestOrder_CancelOrder(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	ctx := context.Background()

	// Cancel the order
	t.Log("Cancelling test order...")
	err := testClient.Clob.CancelOrder(ctx, "0xa80b9b743d0bfdb225c1ecfe80e55b1af2d4e78817e3516291a86caf8af60149")
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}

	t.Log("✓ CancelOrder: Order cancelled successfully")
}

func TestOrder_MarketBuyAndLimitSell(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	ctx := context.Background()

	// Cancel the order
	// Create Market Order
	// t.Log("Buying test market order...")
	// Use the YES token for "Will the US invade Venezuela in 2025?"
	testTokenID := "54544668003738758433824606108816646836259157102970789856801480211062827461038"

	// Ensure we have a valid tick size for this market
	tickSize, err := testClient.Clob.GetTickSize(ctx, testTokenID)
	if err != nil {
		t.Logf("Failed to get tick size, defaulting to 0.01: %v", err)
		tickSize = "0.01"
	}

	// orderResp, err := testClient.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
	// 	TokenID:    testTokenID,
	// 	Amount:     1.0, // Spending 1 USDC (for Buy)
	// 	Side:       clob.SideBuy,
	// 	FeeRateBps: 0,
	// }, tickSize, false)

	// if err != nil {
	// 	t.Fatalf("Create Buy Market Order failed: %v", err)
	// }

	// if orderResp == nil || orderResp.OrderID == "" {
	// 	t.Fatal("Expected valid order response with OrderID")
	// }

	// t.Logf("✓ Create Buy Market Order: OrderID=%s, Status=%s", orderResp.OrderID, orderResp.Status)

	// // loop order status
	// for {
	// 	orderResp, err := testClient.Clob.GetOrder(ctx, orderResp.OrderID)
	// 	if err != nil {
	// 		t.Fatalf("GetOrder failed: %v", err)
	// 	}

	// 	if orderResp.Status == clob.OrderStatusMatched {
	// 		break
	// 	}

	// 	time.Sleep(2 * time.Second)
	// }

	// post limit sell order
	orderResp, err := testClient.Clob.CreateOrder(ctx, clob.UserOrderParams{
		TokenID:    testTokenID,
		Price:      0.15,
		Size:       6.73,
		Side:       clob.SideSell,
		FeeRateBps: 0,
	}, tickSize, false)

	if err != nil {
		t.Fatalf("Create Sell Limit Order failed: %v", err)
	}

	if orderResp == nil || orderResp.OrderID == "" {
		t.Fatal("Expected valid order response with OrderID")
	}

	t.Logf("✓ Create Sell Limit Order: OrderID=%s, Status=%s", orderResp.OrderID, orderResp.Status)

	// loop order status
	for {
		orderResp, err := testClient.Clob.GetOrder(ctx, orderResp.OrderID)
		if err != nil {
			t.Fatalf("GetOrder failed: %v", err)
		}

		if orderResp.Status == clob.OrderStatusMatched {
			break
		}

		time.Sleep(2 * time.Second)
	}

	t.Logf("✓ Sell Limit Order Matched: OrderID=%s, Status=%s", orderResp.OrderID, orderResp.Status)
}

func TestOrder_MarketOrder_CreateAndSell(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	ctx := context.Background()

	// Cancel the order
	// Create Market Order
	// t.Log("Buying test market order...")
	// Use the YES token for "Will the US invade Venezuela in 2025?"
	testTokenID := "54544668003738758433824606108816646836259157102970789856801480211062827461038"

	// Ensure we have a valid tick size for this market
	tickSize, err := testClient.Clob.GetTickSize(ctx, testTokenID)
	if err != nil {
		t.Logf("Failed to get tick size, defaulting to 0.01: %v", err)
		tickSize = "0.01"
	}

	// orderResp, err := testClient.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
	// 	TokenID:    testTokenID,
	// 	Amount:     1.0, // Spending 1 USDC (for Buy)
	// 	Side:       clob.SideBuy,
	// 	FeeRateBps: 0,
	// }, tickSize, false)

	// if err != nil {
	// 	t.Fatalf("Create Buy Market Order failed: %v", err)
	// }

	// if orderResp == nil || orderResp.OrderID == "" {
	// 	t.Fatal("Expected valid order response with OrderID")
	// }

	// t.Logf("✓ Create Buy Market Order: OrderID=%s, Status=%s", orderResp.OrderID, orderResp.Status)

	// // Wait order filled
	// time.Sleep(2 * time.Second)

	// sell now
	t.Log("Selling test market order...")
	orderResp, err := testClient.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
		TokenID:    testTokenID,
		Amount:     0.42, // sell shares
		Side:       clob.SideSell,
		FeeRateBps: 0,
	}, tickSize, false)

	if err != nil {
		t.Fatalf("Create Sell Market Order failed: %v", err)
	}

	if orderResp == nil || orderResp.OrderID == "" {
		t.Fatal("Expected valid order response with OrderID")
	}

	t.Logf("✓ Create Sell Market Order: OrderID=%s, Status=%s", orderResp.OrderID, orderResp.Status)

	// Wait order filled
	time.Sleep(2 * time.Second)

}

func TestOrder_CreateAndCancelOrder(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	if testing.Short() {
		t.Skip("Skipping real order test in short mode")
	}

	ctx := context.Background()

	// Create a small test order
	// Using a price unlikely to be matched (0.01 for YES token)
	t.Log("Creating test order...")
	orderResp, err := testClient.Clob.CreateOrder(ctx, clob.UserOrderParams{
		TokenID: testTokenID,
		Price:   0.01,
		Size:    1.0,
		Side:    clob.SideBuy,
	}, "", false)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if orderResp == nil || orderResp.OrderID == "" {
		t.Fatal("Expected valid order response with OrderID")
	}

	t.Logf("✓ CreateOrder: OrderID=%s, Status=%s", orderResp.OrderID, orderResp.Status)

	// Wait a moment for order to be processed
	time.Sleep(1 * time.Second)

	// Get the order details
	t.Log("Fetching order details...")
	order, err := testClient.Clob.GetOrder(ctx, orderResp.OrderID)
	if err != nil {
		t.Logf("Warning: GetOrder failed: %v", err)
	} else {
		t.Logf("✓ GetOrder: OrderID=%s, Status=%s", order.OrderID, order.Status)
	}

	// Cancel the order
	t.Log("Cancelling test order...")
	err = testClient.Clob.CancelOrder(ctx, orderResp.OrderID)
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}

	t.Log("✓ CancelOrder: Order cancelled successfully")
}

// ========================================
// Account Tests (account.go)
// ========================================

func TestAccount_GetBalanceAllowance(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	ctx := context.Background()

	balance, err := testClient.Clob.GetBalanceAllowance(ctx, clob.BalanceAllowanceParams{
		AssetType: "USDC",
	})
	if err != nil {
		t.Fatalf("GetBalanceAllowance failed: %v", err)
	}

	if balance == nil {
		t.Fatal("Expected non-nil balance response")
	}

	t.Logf("✓ GetBalanceAllowance: Balance=%s, Allowance=%s", balance.Balance, balance.Allowance)
}

func TestAccount_GetClosedOnlyMode(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	ctx := context.Background()

	status, err := testClient.Clob.GetClosedOnlyMode(ctx)
	if err != nil {
		t.Fatalf("GetClosedOnlyMode failed: %v", err)
	}

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	t.Logf("✓ GetClosedOnlyMode: ClosedOnly=%v", status.ClosedOnly)
}

// ========================================
// Batch Operations Tests
// ========================================

func TestOrder_CancelAllOrders(t *testing.T) {
	if testCreds == nil {
		t.Skip("Skipping test: requires valid credentials in .env")
	}

	if testing.Short() {
		t.Skip("Skipping batch operation test in short mode")
	}

	ctx := context.Background()

	err := testClient.Clob.CancelAllOrders(ctx)
	if err != nil {
		t.Fatalf("CancelAllOrders failed: %v", err)
	}

	t.Log("✓ CancelAllOrders: All orders cancelled successfully")
}
