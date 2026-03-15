package data

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataClient_GetPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/positions", r.URL.Path)
		assert.Equal(t, "user123", r.URL.Query().Get("user"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "CASHPNL", r.URL.Query().Get("sortBy"))
		
		positions := []Position{
			{
				Asset:       "0x123",
				ConditionID: "0xcond",
				Size:        100.0,
				AvgPrice:    0.5,
				Title:       "Test Market",
			},
		}
		json.NewEncoder(w).Encode(positions)
	}))
	defer server.Close()

	client := NewDataClient(http.DefaultClient)
	client.BaseURL = server.URL

	params := PositionsParams{
		Limit:  10,
		SortBy: "CASHPNL",
	}
	positions, err := client.GetPositions("user123", params)
	assert.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.Equal(t, "0x123", positions[0].Asset)
}

func TestDataClient_GetActivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/activity", r.URL.Path)
		assert.Equal(t, "user123", r.URL.Query().Get("user"))
		assert.Equal(t, "TRADE", r.URL.Query().Get("type"))
		
		activities := []Activity{
			{
				Type:      "TRADE",
				Side:      "BUY",
				Size:      10.0,
				Timestamp: 1234567890,
			},
		}
		json.NewEncoder(w).Encode(activities)
	}))
	defer server.Close()

	client := NewDataClient(http.DefaultClient)
	client.BaseURL = server.URL

	params := ActivityParams{
		Type: "TRADE",
		Limit: 10,
	}
	activities, err := client.GetActivity("user123", params)
	assert.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "TRADE", activities[0].Type)
}

func TestDataClient_GetTrades(t *testing.T) {
	// Test client-side filtering
	startTs := int64(1000)
	endTs := int64(2000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trades", r.URL.Path)
		assert.Equal(t, "cond123", r.URL.Query().Get("market"))
		
		trades := []DataTrade{
			{Timestamp: 500, Price: 0.1},  // Should be filtered out
			{Timestamp: 1500, Price: 0.2}, // Should match
			{Timestamp: 2500, Price: 0.3}, // Should be filtered out
		}
		json.NewEncoder(w).Encode(trades)
	}))
	defer server.Close()

	client := NewDataClient(http.DefaultClient)
	client.BaseURL = server.URL

	params := TradesParams{
		Market:         "cond123",
		StartTimestamp: startTs,
		EndTimestamp:   endTs,
		Limit:          10,
	}
	trades, err := client.GetTrades(params)
	assert.NoError(t, err)
	assert.Len(t, trades, 1)
	assert.Equal(t, 0.2, trades[0].Price)
}

func TestDataClient_GetLeaderboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/leaderboard", r.URL.Path)
		assert.Equal(t, "WEEK", r.URL.Query().Get("timePeriod"))
		
		entries := []APILeaderboardEntry{
			{
				ProxyWallet: "0xABC",
				Rank:        1.0, // Number
				Pnl:         1000.0,
			},
			{
				ProxyWallet: "0xDEF",
				Rank:        "2", // String
				Pnl:         500.0,
			},
		}
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()

	client := NewDataClient(http.DefaultClient)
	client.BaseURL = server.URL

	params := LeaderboardParams{
		TimePeriod: "WEEK",
		Limit:      10,
	}
	result, err := client.GetLeaderboard(params)
	assert.NoError(t, err)
	assert.Len(t, result.Entries, 2)
	assert.Equal(t, 1, result.Entries[0].Rank)
	assert.Equal(t, 2, result.Entries[1].Rank)
}

func TestDataClient_Integration_RealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDataClient(http.DefaultClient)
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// Use a known active address (e.g., Polymarket top trader or the user's Safe address if active)
	// Using a top trader from leaderboard to ensure data exists, or just the user's safe address
	// User's Safe Address: 0x275165e1492406133e1C8F064f537Ca7F2bEf0A0
	targetAddress := "0x275165e1492406133e1C8F064f537Ca7F2bEf0A0" 

	t.Run("GetPositions", func(t *testing.T) {
		positions, err := client.GetPositions(targetAddress, PositionsParams{
			Limit: 5,
		})
		if err != nil {
			t.Fatalf("Failed to get positions: %v", err)
		}
		t.Logf("Fetched %d positions", len(positions))
		for _, p := range positions {
			t.Logf("- %s: %f shares (PnL: %f)", p.Title, p.Size, p.CashPnl)
		}
	})

	t.Run("GetActivity", func(t *testing.T) {
		activities, err := client.GetActivity(targetAddress, ActivityParams{
			Limit: 5,
		})
		if err != nil {
			t.Fatalf("Failed to get activity: %v", err)
		}
		t.Logf("Fetched %d activities", len(activities))
		for _, a := range activities {
			t.Logf("- %d: %s %f (Tx: %s)", a.Timestamp, a.Type, a.Size, a.TransactionHash)
		}
	})

	t.Run("GetTrades", func(t *testing.T) {
		// Use a known market or just fetch recent trades
		// US-Iran Nuclear Deal: 0x182390641d3b1b47cc64274b9da290efd04221c586651ba190880713da6347d9
		conditionID := "0x182390641d3b1b47cc64274b9da290efd04221c586651ba190880713da6347d9"
		
		trades, err := client.GetTrades(TradesParams{
			Market: conditionID,
			Limit:  5,
		})
		if err != nil {
			t.Fatalf("Failed to get trades: %v", err)
		}
		t.Logf("Fetched %d trades for market %s", len(trades), conditionID)
		for _, tr := range trades {
			t.Logf("- %s: %f @ %f (Side: %s)", time.UnixMilli(tr.Timestamp), tr.Size, tr.Price, tr.Side)
		}
	})

	t.Run("GetLeaderboard", func(t *testing.T) {
		lb, err := client.GetLeaderboard(LeaderboardParams{
			TimePeriod: "WEEK",
			Limit:      5,
		})
		if err != nil {
			t.Fatalf("Failed to get leaderboard: %v", err)
		}
		t.Logf("fetched %d leaderboard entries", len(lb.Entries))
		for _, e := range lb.Entries {
			t.Logf("#%d %s (PnL: %f)", e.Rank, e.UserName, e.Pnl)
		}
	})
}
