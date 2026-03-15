package clob

import (
	"context"
	"fmt"
	"strconv"
)

// Market data endpoints
const (
	EndpointTime                      = "/time"
	EndpointMarkets                   = "/markets"
	EndpointMarket                    = "/markets/"
	EndpointSimplifiedMarkets         = "/simplified-markets"
	EndpointSamplingMarkets           = "/sampling-markets"
	EndpointSamplingSimplifiedMarkets = "/sampling-simplified-markets"
	EndpointOrderBook                 = "/book"
	EndpointOrderBooks                = "/books"
	EndpointMidpoint                  = "/midpoint"
	EndpointMidpoints                 = "/midpoints"
	EndpointPrice                     = "/price"
	EndpointPrices                    = "/prices"
	EndpointSpread                    = "/spread"
	EndpointSpreads                   = "/spreads"
	EndpointLastTradePrice            = "/last-trade-price"
	EndpointLastTradesPrices          = "/last-trades-prices"
	EndpointPricesHistory             = "/prices-history"
	EndpointTrades                    = "/data/trades"
	EndpointMarketTradesEvents        = "/live-activity/events/"
)

// Pagination constants
const (
	InitialCursor = ""
	EndCursor     = "LTE="
)

// GetServerTime returns the current server timestamp in seconds
func (c *ClobClient) GetServerTime(ctx context.Context) (int64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointTime)

	var timestamp int64
	if err := c.get(ctx, url, nil, &timestamp, false); err != nil {
		return 0, fmt.Errorf("failed to get server time: %w", err)
	}

	return timestamp, nil
}

// GetMarket fetches a single market by condition ID
func (c *ClobClient) GetMarket(ctx context.Context, conditionID string) (*Market, error) {
	url := fmt.Sprintf("%s%s%s", c.baseURL, EndpointMarket, conditionID)

	var market Market
	if err := c.get(ctx, url, nil, &market, false); err != nil {
		return nil, fmt.Errorf("failed to get market %s: %w", conditionID, err)
	}

	return &market, nil
}

// GetMarkets fetches a paginated list of markets
func (c *ClobClient) GetMarkets(ctx context.Context, nextCursor string) (*PaginationPayload, error) {
	if nextCursor == "" {
		nextCursor = InitialCursor
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointMarkets)
	params := map[string]string{"next_cursor": nextCursor}

	var result PaginationPayload
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return nil, fmt.Errorf("failed to get markets: %w", err)
	}

	return &result, nil
}

// GetActiveMarkets fetches a paginated list of active markets
func (c *ClobClient) GetActiveMarkets(ctx context.Context, nextCursor string) (*PaginationPayload, error) {
	if nextCursor == "" {
		nextCursor = InitialCursor
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointMarkets)
	params := map[string]string{
		"next_cursor": nextCursor,
		"active":      "true",
		"closed":      "false",
	}

	var result PaginationPayload
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return nil, fmt.Errorf("failed to get active markets: %w", err)
	}

	return &result, nil
}

// GetSimplifiedMarkets fetches a paginated list of simplified market data
func (c *ClobClient) GetSimplifiedMarkets(ctx context.Context, nextCursor string) (*PaginationPayload, error) {
	if nextCursor == "" {
		nextCursor = InitialCursor
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointSimplifiedMarkets)
	params := map[string]string{"next_cursor": nextCursor}

	var result PaginationPayload
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return nil, fmt.Errorf("failed to get simplified markets: %w", err)
	}

	return &result, nil
}

// GetSamplingMarkets fetches a sample of markets for quick overview
func (c *ClobClient) GetSamplingMarkets(ctx context.Context, nextCursor string) (*PaginationPayload, error) {
	if nextCursor == "" {
		nextCursor = InitialCursor
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointSamplingMarkets)
	params := map[string]string{"next_cursor": nextCursor}

	var result PaginationPayload
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return nil, fmt.Errorf("failed to get sampling markets: %w", err)
	}

	return &result, nil
}

// GetOrderBook fetches the order book for a specific token
func (c *ClobClient) GetOrderBook(ctx context.Context, tokenID string) (*OrderBookSummary, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointOrderBook)
	params := map[string]string{"token_id": tokenID}

	var orderbook OrderBookSummary
	if err := c.get(ctx, url, params, &orderbook, false); err != nil {
		return nil, fmt.Errorf("failed to get order book for token %s: %w", tokenID, err)
	}

	return &orderbook, nil
}

// GetOrderBooks fetches order books for multiple tokens in batch
func (c *ClobClient) GetOrderBooks(ctx context.Context, params []BookParams) ([]*OrderBookSummary, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointOrderBooks)

	var orderbooks []*OrderBookSummary
	if err := c.post(ctx, url, params, &orderbooks, false); err != nil {
		return nil, fmt.Errorf("failed to get order books: %w", err)
	}

	return orderbooks, nil
}

// GetMidpoint fetches the midpoint price for a token
func (c *ClobClient) GetMidpoint(ctx context.Context, tokenID string) (float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointMidpoint)
	params := map[string]string{"token_id": tokenID}

	var result map[string]interface{}
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return 0, fmt.Errorf("failed to get midpoint for token %s: %w", tokenID, err)
	}

	// string to float64
	if mid, ok := result["mid"].(string); ok {
		return strconv.ParseFloat(mid, 64)
	}

	return 0, fmt.Errorf("invalid midpoint response format")
}

// GetMidpoints fetches midpoints for multiple tokens in batch
func (c *ClobClient) GetMidpoints(ctx context.Context, params []BookParams) ([]float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointMidpoints)

	var results []map[string]interface{}
	if err := c.post(ctx, url, params, &results, false); err != nil {
		return nil, fmt.Errorf("failed to get midpoints: %w", err)
	}

	midpoints := make([]float64, len(results))
	for i, result := range results {
		if mid, ok := result["mid"].(float64); ok {
			midpoints[i] = mid
		}
	}

	return midpoints, nil
}

// GetPrice fetches the best price for a token and side
func (c *ClobClient) GetPrice(ctx context.Context, tokenID string, side Side) (float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointPrice)
	params := map[string]string{
		"token_id": tokenID,
		"side":     string(side),
	}

	var result map[string]interface{}
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return 0, fmt.Errorf("failed to get price for token %s: %w", tokenID, err)
	}

	if price, ok := result["price"].(string); ok {
		return strconv.ParseFloat(price, 64)
	}

	return 0, fmt.Errorf("invalid price response format")
}

// GetPrices fetches prices for multiple tokens in batch
func (c *ClobClient) GetPrices(ctx context.Context, params []BookParams) ([]float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointPrices)

	var results []map[string]interface{}
	if err := c.post(ctx, url, params, &results, false); err != nil {
		return nil, fmt.Errorf("failed to get prices: %w", err)
	}

	prices := make([]float64, len(results))
	for i, result := range results {
		if price, ok := result["price"].(string); ok {
			prices[i], _ = strconv.ParseFloat(price, 64)
		}
	}

	return prices, nil
}

// GetSpread fetches the bid-ask spread for a token
func (c *ClobClient) GetSpread(ctx context.Context, tokenID string) (float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointSpread)
	params := map[string]string{"token_id": tokenID}

	var result map[string]interface{}
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return 0, fmt.Errorf("failed to get spread for token %s: %w", tokenID, err)
	}

	if spread, ok := result["spread"].(string); ok {
		return strconv.ParseFloat(spread, 64)
	}

	return 0, fmt.Errorf("invalid spread response format")
}

// GetSpreads fetches spreads for multiple tokens in batch
func (c *ClobClient) GetSpreads(ctx context.Context, params []BookParams) ([]float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointSpreads)

	var results []map[string]interface{}
	if err := c.post(ctx, url, params, &results, false); err != nil {
		return nil, fmt.Errorf("failed to get spreads: %w", err)
	}

	spreads := make([]float64, len(results))
	for i, result := range results {
		if spread, ok := result["spread"].(string); ok {
			spreads[i], _ = strconv.ParseFloat(spread, 64)
		}
	}

	return spreads, nil
}

// GetLastTradePrice fetches the last trade price for a token
func (c *ClobClient) GetLastTradePrice(ctx context.Context, tokenID string) (float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointLastTradePrice)
	params := map[string]string{"token_id": tokenID}

	var result map[string]interface{}
	if err := c.get(ctx, url, params, &result, false); err != nil {
		return 0, fmt.Errorf("failed to get last trade price for token %s: %w", tokenID, err)
	}

	if price, ok := result["price"].(string); ok {
		return strconv.ParseFloat(price, 64)
	}

	return 0, fmt.Errorf("invalid last trade price response format")
}

// GetLastTradesPrices fetches last trade prices for multiple tokens in batch
func (c *ClobClient) GetLastTradesPrices(ctx context.Context, params []BookParams) ([]float64, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointLastTradesPrices)

	var results []map[string]interface{}
	if err := c.post(ctx, url, params, &results, false); err != nil {
		return nil, fmt.Errorf("failed to get last trades prices: %w", err)
	}

	prices := make([]float64, len(results))
	for i, result := range results {
		if price, ok := result["price"].(float64); ok {
			prices[i] = price
		}
	}

	return prices, nil
}

// GetPricesHistory fetches historical prices for a market
func (c *ClobClient) GetPricesHistory(ctx context.Context, params PriceHistoryFilterParams) ([]MarketPrice, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointPricesHistory)

	// Convert params to query string map
	queryParams := make(map[string]string)
	if params.Market != "" {
		queryParams["market"] = params.Market
	}
	if params.Interval != "" {
		queryParams["interval"] = params.Interval
	}
	if params.StartTS > 0 {
		queryParams["startTs"] = fmt.Sprintf("%d", params.StartTS)
	}
	if params.EndTS > 0 {
		queryParams["endTs"] = fmt.Sprintf("%d", params.EndTS)
	}
	if params.Fidelity > 0 {
		queryParams["fidelity"] = fmt.Sprintf("%d", params.Fidelity)
	}

	var priceHistory *MarketPriceHistory
	if err := c.get(ctx, url, queryParams, &priceHistory, false); err != nil {
		return nil, fmt.Errorf("failed to get prices history: %w", err)
	}

	return priceHistory.History, nil
}

// GetTrades fetches trade history
func (c *ClobClient) GetTrades(ctx context.Context, params TradeParams) ([]*Trade, error) {
	// This endpoint requires L2 authentication
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointTrades)

	// Convert params to query string map
	queryParams := make(map[string]string)
	if params.Maker != "" {
		queryParams["maker"] = params.Maker
	}
	if params.Market != "" {
		queryParams["market"] = params.Market
	}
	if params.Asset != "" {
		queryParams["asset"] = params.Asset
	}

	var resp struct {
		Data []*Trade `json:"data"`
	}
	if err := c.get(ctx, url, queryParams, &resp, true); err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	return resp.Data, nil
}

// GetTradesPaginated fetches trade history with pagination
func (c *ClobClient) GetTradesPaginated(ctx context.Context, params TradeParams, nextCursor string) (*PaginationPayload, error) {
	if nextCursor == "" {
		nextCursor = InitialCursor
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointTrades)

	// Convert params to query string map
	queryParams := make(map[string]string)
	queryParams["next_cursor"] = nextCursor
	if params.Maker != "" {
		queryParams["maker"] = params.Maker
	}
	if params.Market != "" {
		queryParams["market"] = params.Market
	}
	if params.Asset != "" {
		queryParams["asset"] = params.Asset
	}

	var result PaginationPayload
	if err := c.get(ctx, url, queryParams, &result, true); err != nil {
		return nil, fmt.Errorf("failed to get paginated trades: %w", err)
	}

	return &result, nil
}

// GetMarketTradesEvents fetches live activity events for a market
func (c *ClobClient) GetMarketTradesEvents(ctx context.Context, conditionID string) ([]*MarketTradeEvent, error) {
	url := fmt.Sprintf("%s%s%s", c.baseURL, EndpointMarketTradesEvents, conditionID)

	var events []*MarketTradeEvent
	if err := c.get(ctx, url, nil, &events, false); err != nil {
		return nil, fmt.Errorf("failed to get market trade events: %w", err)
	}

	return events, nil
}
