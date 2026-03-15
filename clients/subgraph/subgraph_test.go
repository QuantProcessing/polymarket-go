package subgraph

import (
	"net/http"
	"testing"
)

func TestSubgraphClient_Integration_RealAPI(t *testing.T) {
	// Only run with -v to see logs, or if user explicitly wants to run them.
	// We keep Short() check but user implies they want real tests.
	// If the user runs `go test`, it runs these if not short.
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewSubgraphClient(http.DefaultClient)
	// Use a known active address (e.g. Polymarket top trader or user's proxy)
	// 0x275165e1492406133e1C8F064f537Ca7F2bEf0A0
	targetAddr := "0x275165e1492406133e1C8F064f537Ca7F2bEf0A0"
	knownConditionID := "0x182390641d3b1b47cc64274b9da290efd04221c586651ba190880713da6347d9" // US-Iran Nuclear

	t.Run("GetUserBalances", func(t *testing.T) {
		balances, err := client.GetUserBalances(targetAddr, SubgraphQueryParams{First: IntP(5)})
		if err != nil {
			t.Fatalf("Failed to get balances: %v", err)
		}
		t.Logf("Fetched %d balances", len(balances))
		for _, b := range balances {
			t.Logf("- %s: %s", b.Asset, b.Balance)
		}
	})

	t.Run("GetUserPositions", func(t *testing.T) {
		positions, err := client.GetUserPositions(targetAddr, SubgraphQueryParams{
			First:   IntP(5),
			OrderBy: "id", // Use ID sort to avoid timeout
		})
		if err != nil {
			t.Fatalf("Failed to get positions: %v", err)
		}
		t.Logf("Fetched %d positions", len(positions))
		for _, p := range positions {
			t.Logf("- %s: %s (Realized PnL: %s)", p.TokenID, p.Amount, p.RealizedPnl)
		}
	})

	t.Run("GetSplits", func(t *testing.T) {
		splits, err := client.GetSplits(targetAddr, SubgraphQueryParams{First: IntP(5)})
		if err != nil {
			t.Fatalf("Failed to get splits: %v", err)
		}
		t.Logf("Fetched %d splits", len(splits))
		for _, s := range splits {
			t.Logf("- %s: %s (Condition: %s)", s.Timestamp, s.Amount, s.Condition)
		}
	})

	t.Run("GetMarketOpenInterest", func(t *testing.T) {
		oi, err := client.GetMarketOpenInterest(knownConditionID)
		if err != nil {
			t.Fatalf("Failed to get market OI: %v", err)
		}
		if oi == nil {
			t.Logf("No OI found for condition %s", knownConditionID)
		} else {
			t.Logf("Market OI for %s: %s", oi.ID, oi.Amount)
		}
	})

	t.Run("GetTopMarketsByOI", func(t *testing.T) {
		ois, err := client.GetTopMarketsByOI(SubgraphQueryParams{
			First:   IntP(5),
			OrderBy: "id", // Use ID sort to avoid timeout
		})
		if err != nil {
			t.Fatalf("Failed to get top markets by OI: %v", err)
		}
		t.Logf("Fetched %d markets", len(ois))
		for _, o := range ois {
			t.Logf("- %s: %s", o.ID, o.Amount)
		}
	})

	t.Run("GetOrderFilledEvents", func(t *testing.T) {
		events, err := client.GetOrderFilledEvents(SubgraphQueryParams{First: IntP(5)})
		if err != nil {
			t.Fatalf("Failed to get order filled events: %v", err)
		}
		t.Logf("Fetched %d events", len(events))
		for _, e := range events {
			t.Logf("- %s: %s (Maker: %s)", e.Timestamp, e.MakerAmountFilled, e.Maker)
		}
	})
}
