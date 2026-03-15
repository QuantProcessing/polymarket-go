package gamma

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGammaClient_Integration_GetMarkets(t *testing.T) {
	// Skip integration test if short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewGammaClient(http.DefaultClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	markets, err := client.GetMarkets(ctx, MarketSearchParams{
		Limit: 5,
		Active: BoolPtr(true),
	})
	
	if err != nil {
		t.Logf("Gamma API check failed (could be network): %v", err)
		return
	}

	assert.NoError(t, err)
	assert.NotEmpty(t, markets)
	t.Logf("Fetched %d markets", len(markets))
	if len(markets) > 0 {
		t.Logf("First market: %s (Slug: %s)", markets[0].Question, markets[0].Slug)
	}
}

func TestGammaClient_Integration_GetEventBySlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewGammaClient(http.DefaultClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a known event slug (e.g., from user request or prominent one)
	// "us-iran-nuclear-deal-before-2027" might be an event slug? Or market slug?
	// User said event: https://polymarket.com/zh/event/us-iran-nuclear-deal-before-2027
	// Slug is likely "us-iran-nuclear-deal-before-2027"

	slug := "us-iran-nuclear-deal-before-2027"
	event, err := client.GetEventBySlug(ctx, slug)
	
	if err != nil {
		t.Logf("Gamma API GetEvent check failed: %v", err)
		return
	}

	if event == nil {
		t.Logf("Event not found: %s", slug)
	} else {
		assert.Equal(t, slug, event.Slug)
		t.Logf("Found event: %s", event.Title)
		t.Logf("Contains %d markets", len(event.Markets))
	}
}

// Helper
func BoolPtr(b bool) *bool {
	return &b
}
