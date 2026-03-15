package gamma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// Gamma API Base URL
const GammaAPIBase = "https://gamma-api.polymarket.com"

// GammaMarket represents a market from the Gamma API
type GammaMarket struct {
	ID                       string          `json:"id"`
	ConditionID              string          `json:"conditionId"`
	Slug                     string          `json:"slug"`
	Question                 string          `json:"question"`
	Description              string          `json:"description,omitempty"`
	// API returns these as stringified JSON arrays sometimes?
	// or maybe it's []string but my previous assumption was wrong?
	// The error `cannot unmarshal string into ... []string` confirms it returns a string.
	Outcomes                 string          `json:"outcomes"`
	OutcomePrices            string          `json:"outcomePrices"` 
	Volume                   decimal.Decimal `json:"volume"`
	Volume24hr               decimal.Decimal `json:"volume24hr,omitempty"`
	Volume1wk                decimal.Decimal `json:"volume1wk,omitempty"`
	Liquidity                decimal.Decimal `json:"liquidity"`
	Spread                   decimal.Decimal `json:"spread,omitempty"`
	OneDayPriceChange        decimal.Decimal `json:"oneDayPriceChange,omitempty"`
	OneWeekPriceChange       decimal.Decimal `json:"oneWeekPriceChange,omitempty"`
	LastTradePrice           decimal.Decimal `json:"lastTradePrice,omitempty"`
	BestBid                  decimal.Decimal `json:"bestBid,omitempty"`
	BestAsk                  decimal.Decimal `json:"bestAsk,omitempty"`
	EndDate                  *time.Time      `json:"endDate"`
	CreatedAt                *time.Time      `json:"createdAt,omitempty"`
	StartDate                *time.Time      `json:"startDate,omitempty"`
	AcceptingOrdersTimestamp *time.Time      `json:"acceptingOrdersTimestamp,omitempty"`
	Active                   bool            `json:"active"`
	Closed                   bool            `json:"closed"`
	Image                    string          `json:"image,omitempty"`
	Icon                     string          `json:"icon,omitempty"`
	Tags                     []GammaTag      `json:"tags,omitempty"` // API might return strings or objects? TS says strings but also defines GammaTag interface. Checking TS implem, it says `tags?: string[]`.
	// Let's stick to []string for now based on GammaMarket interface in TS which says `tags?: string[]`
	// Wait, TS says `tags?: string[]` in GammaMarket, so we use []string
	TagStrings []string `json:"-"` 
	
	// ClobTokenIds is stringified JSON array in API response
	ClobTokenIds string `json:"clobTokenIds,omitempty"`
}

// GammaToken represents a token for an outcome
type GammaToken struct {
	TokenID string
	Outcome string
	Price   decimal.Decimal
}

// Tokens helper returns structured token info
func (m GammaMarket) Tokens() []GammaToken {
	tokens := make([]GammaToken, 0)
	// Parse outcomes
	var outcomes []string
	if err := json.Unmarshal([]byte(m.Outcomes), &outcomes); err != nil {
		// Fallback or log?
	}
	// Parse prices
	var priceFloats []float64
	json.Unmarshal([]byte(m.OutcomePrices), &priceFloats)
	
	// Parse TokenIds
	var tokenIds []string
	json.Unmarshal([]byte(m.ClobTokenIds), &tokenIds)

	for i, id := range tokenIds {
		outcome := ""
		if i < len(outcomes) {
			outcome = outcomes[i]
		}
		var price decimal.Decimal
		if i < len(priceFloats) {
			price = decimal.NewFromFloat(priceFloats[i])
		}
		tokens = append(tokens, GammaToken{
			TokenID: id,
			Outcome: outcome,
			Price:   price,
		})
	}
	return tokens
}

// GammaTag represents detailed tag info (from separate endpoint usually)
// But GammaMarket has `tags: string[]`.

// GammaEvent represents an event grouping
type GammaEvent struct {
	ID          string        `json:"id"`
	Slug        string        `json:"slug"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Markets     []GammaMarket `json:"markets"`
	StartDate   *time.Time    `json:"startDate,omitempty"`
	EndDate     *time.Time    `json:"endDate,omitempty"`
	Image       string        `json:"image,omitempty"`
}

// MarketSearchParams parameters for querying markets
type MarketSearchParams struct {
	Slug        string
	ConditionID string
	Active      *bool
	Closed      *bool
	Limit       int
	Offset      int
	Order       string
	Ascending   *bool
	Tag         string
}

// EventSearchParams parameters for querying events
type EventSearchParams struct {
	Slug   string
	Active *bool
	Limit  int
}

// GammaClient interacts with the Gamma API
type GammaClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewGammaClient creates a new GammaClient
func NewGammaClient(httpClient *http.Client) *GammaClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &GammaClient{
		httpClient: httpClient,
		baseURL:    GammaAPIBase,
	}
}

// GetMarkets fetches markets with filters
func (c *GammaClient) GetMarkets(ctx context.Context, params MarketSearchParams) ([]GammaMarket, error) {
	u, err := url.Parse(fmt.Sprintf("%s/markets", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if params.Slug != "" {
		q.Set("slug", params.Slug)
	}
	if params.ConditionID != "" {
		q.Set("condition_id", params.ConditionID)
	}
	if params.Active != nil {
		q.Set("active", strconv.FormatBool(*params.Active))
	}
	if params.Closed != nil {
		q.Set("closed", strconv.FormatBool(*params.Closed))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Order != "" {
		q.Set("order", params.Order)
	}
	if params.Ascending != nil {
		q.Set("ascending", strconv.FormatBool(*params.Ascending))
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gamma api returned status: %d", resp.StatusCode)
	}

	var markets []GammaMarket
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return markets, nil
}

// GetMarketBySlug fetches a single market by slug
func (c *GammaClient) GetMarketBySlug(ctx context.Context, slug string) (*GammaMarket, error) {
	markets, err := c.GetMarkets(ctx, MarketSearchParams{Slug: slug, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(markets) == 0 {
		return nil, nil
	}
	return &markets[0], nil
}

// GetMarketByConditionID fetches a single market by condition ID
func (c *GammaClient) GetMarketByConditionID(ctx context.Context, conditionID string) (*GammaMarket, error) {
	markets, err := c.GetMarkets(ctx, MarketSearchParams{ConditionID: conditionID, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(markets) == 0 {
		return nil, nil
	}
	return &markets[0], nil
}

// GetEvents fetches events with filters
func (c *GammaClient) GetEvents(ctx context.Context, params EventSearchParams) ([]GammaEvent, error) {
	u, err := url.Parse(fmt.Sprintf("%s/events", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if params.Slug != "" {
		q.Set("slug", params.Slug)
	}
	if params.Active != nil {
		q.Set("active", strconv.FormatBool(*params.Active))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gamma api returned status: %d", resp.StatusCode)
	}

	var events []GammaEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return events, nil
}

// GetEventBySlug fetches a single event by slug
func (c *GammaClient) GetEventBySlug(ctx context.Context, slug string) (*GammaEvent, error) {
	events, err := c.GetEvents(ctx, EventSearchParams{Slug: slug, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	return &events[0], nil
}

// GetEventByID fetches a single event by ID
func (c *GammaClient) GetEventByID(ctx context.Context, id string) (*GammaEvent, error) {
	u, err := url.Parse(fmt.Sprintf("%s/events/%s", c.baseURL, id))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gamma api returned status: %d", resp.StatusCode)
	}

	var event GammaEvent
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &event, nil
}

// Helper types for JSON decoding custom time format if needed. 
// Standard time.Time JSON unmarshalling works for RFC3339.
// Gamma API usually returns ISO8601/RFC3339 strings.

// Tag struct if we need it later
type GammaTag struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
}
