package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const DataAPIBase = "https://data-api.polymarket.com"

// DataClient interacts with Polymarket Data API
type DataClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewDataClient creates a new DataClient
func NewDataClient(httpClient *http.Client) *DataClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	// Check for proxy in environment
	proxyURL := os.Getenv("PROXY")
	if proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err == nil {
			// If existing transport is *http.Transport, update it. Otherwise create new.
			if t, ok := httpClient.Transport.(*http.Transport); ok {
				t.Proxy = http.ProxyURL(parsedURL)
			} else {
				httpClient.Transport = &http.Transport{
					Proxy: http.ProxyURL(parsedURL),
				}
			}
		}
	}

	return &DataClient{
		BaseURL:    DataAPIBase,
		HTTPClient: httpClient,
	}
}

// GetPositions retrieves positions for a wallet
func (c *DataClient) GetPositions(address string, params PositionsParams) ([]Position, error) {
	q := url.Values{}
	q.Set("user", address)

	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.SortBy != "" {
		q.Set("sortBy", params.SortBy)
	}
	if params.SortDirection != "" {
		q.Set("sortDirection", params.SortDirection)
	}
	for _, m := range params.Market {
		q.Add("market", m)
	}
	for _, id := range params.EventID {
		q.Add("eventId", strconv.Itoa(id))
	}
	if params.SizeThreshold > 0 {
		q.Set("sizeThreshold", fmt.Sprintf("%f", params.SizeThreshold))
	}
	if params.Redeemable != nil {
		q.Set("redeemable", strconv.FormatBool(*params.Redeemable))
	}
	if params.Mergeable != nil {
		q.Set("mergeable", strconv.FormatBool(*params.Mergeable))
	}
	if params.Title != "" {
		q.Set("title", params.Title)
	}

	var positions []Position
	err := c.doRequest("GET", "/positions", q, &positions)
	return positions, err
}

// GetClosedPositions retrieves closed positions for a wallet
func (c *DataClient) GetClosedPositions(address string, params ClosedPositionsParams) ([]ClosedPosition, error) {
	q := url.Values{}
	q.Set("user", address)

	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	for _, m := range params.Market {
		q.Add("market", m)
	}
	for _, id := range params.EventID {
		q.Add("eventId", strconv.Itoa(id))
	}
	if params.Title != "" {
		q.Set("title", params.Title)
	}
	if params.SortBy != "" {
		q.Set("sortBy", params.SortBy)
	}
	if params.SortDirection != "" {
		q.Set("sortDirection", params.SortDirection)
	}

	var positions []ClosedPosition
	err := c.doRequest("GET", "/closed-positions", q, &positions)
	return positions, err
}

// GetActivity retrieves activity for a wallet
func (c *DataClient) GetActivity(address string, params ActivityParams) ([]Activity, error) {
	q := url.Values{}
	q.Set("user", address)
	
	limit := params.Limit
	if limit == 0 {
		limit = 100
	}
	q.Set("limit", strconv.Itoa(limit))

	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Start > 0 {
		q.Set("start", strconv.FormatInt(params.Start, 10))
	}
	if params.End > 0 {
		q.Set("end", strconv.FormatInt(params.End, 10))
	}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	if params.Side != "" {
		q.Set("side", params.Side)
	}
	for _, m := range params.Market {
		q.Add("market", m)
	}
	for _, id := range params.EventID {
		q.Add("eventId", strconv.Itoa(id))
	}
	if params.SortBy != "" {
		q.Set("sortBy", params.SortBy)
	}
	if params.SortDirection != "" {
		q.Set("sortDirection", params.SortDirection)
	}

	var activities []Activity
	err := c.doRequest("GET", "/activity", q, &activities)
	return activities, err
}

// GetTrades retrieves trades
func (c *DataClient) GetTrades(params TradesParams) ([]DataTrade, error) {
	q := url.Values{}
	
	limit := params.Limit
	// Request more if we need to filter client-side (approximation logic from TS)
	if params.StartTimestamp > 0 || params.EndTimestamp > 0 {
		if limit == 0 {
			limit = 500
		}
		reqLimit := limit * 3
		if reqLimit > 1000 {
			reqLimit = 1000
		}
		q.Set("limit", strconv.Itoa(reqLimit))
	} else {
		if limit > 0 {
			q.Set("limit", strconv.Itoa(limit))
		}
	}

	if params.Market != "" {
		q.Set("market", params.Market)
	}
	if params.User != "" {
		q.Set("user", params.User)
	}
	if params.TakerOnly != nil {
		q.Set("takerOnly", strconv.FormatBool(*params.TakerOnly))
	}
	if params.FilterType != "" {
		q.Set("filterType", params.FilterType)
	}
	if params.FilterAmount > 0 {
		q.Set("filterAmount", fmt.Sprintf("%f", params.FilterAmount))
	}
	if params.Side != "" {
		q.Set("side", params.Side)
	}

	var trades []DataTrade
	err := c.doRequest("GET", "/trades", q, &trades)
	if err != nil {
		return nil, err
	}

	// Client-side filtering
	if params.StartTimestamp > 0 || params.EndTimestamp > 0 {
		filtered := make([]DataTrade, 0, len(trades))
		for _, t := range trades {
			if params.StartTimestamp > 0 && t.Timestamp < params.StartTimestamp {
				continue
			}
			if params.EndTimestamp > 0 && t.Timestamp > params.EndTimestamp {
				continue
			}
			filtered = append(filtered, t)
		}
		trades = filtered
	}

	// Apply limit after filtering
	if params.Limit > 0 && len(trades) > params.Limit {
		trades = trades[:params.Limit]
	}

	return trades, nil
}

// GetLeaderboard retrieves leaderboard rankings
func (c *DataClient) GetLeaderboard(params LeaderboardParams) (*LeaderboardResult, error) {
	q := url.Values{}
	
	if params.TimePeriod != "" {
		q.Set("timePeriod", params.TimePeriod)
	}
	if params.OrderBy != "" {
		q.Set("orderBy", params.OrderBy)
	}
	if params.Category != "" {
		q.Set("category", params.Category)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.User != "" {
		q.Set("user", params.User)
	}
	if params.UserName != "" {
		q.Set("userName", params.UserName)
	}

	var apiEntries []APILeaderboardEntry
	err := c.doRequest("GET", "/v1/leaderboard", q, &apiEntries)
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, len(apiEntries))
	for i, e := range apiEntries {
		rank := 0
		switch v := e.Rank.(type) {
		case float64:
			rank = int(v)
		case string:
			fmt.Sscanf(v, "%d", &rank)
		}

		entries[i] = LeaderboardEntry{
			Address:       e.ProxyWallet,
			Rank:          rank,
			Pnl:           e.Pnl,
			Volume:        e.Vol,
			UserName:      e.UserName,
			XUsername:     e.XUsername,
			VerifiedBadge: e.VerifiedBadge,
			ProfileImage:  e.ProfileImage,
		}
	}

	hasMore := false
	if params.Limit > 0 && len(entries) == params.Limit {
		hasMore = true
	}

	return &LeaderboardResult{
		Entries: entries,
		HasMore: hasMore,
		Request: params,
	}, nil
}

// DataAPIError represents Data API specific error structure
type DataAPIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	ErrorMsg   string `json:"error"`
}

func (e *DataAPIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Data API Error %d: %s", e.StatusCode, e.Message)
	}
	if e.ErrorMsg != "" {
		return fmt.Sprintf("Data API Error %d: %s", e.StatusCode, e.ErrorMsg)
	}
	return fmt.Sprintf("Data API Error %d", e.StatusCode)
}

func (c *DataClient) doRequest(method, endpoint string, params url.Values, result interface{}) error {
	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return err
	}
	
	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "PolymarketGoClient/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		var apiErr DataAPIError
		if err := json.Unmarshal(body, &apiErr); err == nil && (apiErr.Message != "" || apiErr.ErrorMsg != "") {
			apiErr.StatusCode = resp.StatusCode
			return &apiErr
		}
		// Fallback error
		return fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to decode response: %w (body: %s)", err, string(body))
		}
	}

	return nil
}
