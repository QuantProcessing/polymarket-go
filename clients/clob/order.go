package clob

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

// Order endpoints
const (
	EndpointPostOrders         = "/orders"
	EndpointCancelOrders       = "/orders"
	EndpointCancelMarketOrders = "/cancel-market-orders"
	EndpointOpenOrders         = "/data/orders"
	EndpointTickSize           = "/tick-size"
	EndpointNegRisk            = "/neg-risk"
	EndpointFeeRate            = "/fee-rate"
)

// Metadata caches (thread-safe)
var (
	tickSizeCache = make(map[string]string)
	negRiskCache  = make(map[string]bool)
	feeRateCache  = make(map[string]int)
	cacheMu       sync.RWMutex
)

// PostOrders creates multiple orders in a single batch
func (c *ClobClient) PostOrders(ctx context.Context, orders []PostOrdersArgs, deferExec bool) (*BatchOrderResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointPostOrders)

	var batchResponse BatchOrderResponse
	if err := c.post(ctx, url, orders, &batchResponse, true); err != nil {
		return nil, fmt.Errorf("failed to post orders: %w", err)
	}

	return &batchResponse, nil
}

// CancelOrders cancels multiple orders by their IDs
func (c *ClobClient) CancelOrders(ctx context.Context, orderIDs []string) (*CancelOrdersResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	// Use DELETE /orders endpoint with orderIDs in request body
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointCancelOrders)

	// Request body: array of orderIDs
	payload := orderIDs

	var response CancelOrdersResponse
	if err := c.delete(ctx, url, payload, &response, true); err != nil {
		return nil, fmt.Errorf("failed to cancel orders: %w", err)
	}

	return &response, nil
}

// CancelMarketOrders cancels all orders for a specific market
func (c *ClobClient) CancelMarketOrders(ctx context.Context, params OrderMarketCancelParams) error {
	if err := c.EnsureAuth(ctx); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointCancelMarketOrders)

	if err := c.delete(ctx, url, params, nil, true); err != nil {
		return fmt.Errorf("failed to cancel market orders: %w", err)
	}

	return nil
}

// GetOpenOrders fetches open orders for the authenticated user
func (c *ClobClient) GetOpenOrders(ctx context.Context, params OpenOrderParams) (*OpenOrdersResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointOpenOrders)

	// Convert params to query map
	queryParams := make(map[string]string)
	if params.Market != "" {
		queryParams["market"] = params.Market
	}
	if params.Asset != "" {
		queryParams["asset"] = params.Asset
	}
	if params.NextCursor != "" {
		queryParams["next_cursor"] = params.NextCursor
	}

	var response OpenOrdersResponse
	if err := c.get(ctx, url, queryParams, &response, true); err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	return &response, nil
}

// GetTickSize fetches the tick size for a token (with caching)
func (c *ClobClient) GetTickSize(ctx context.Context, tokenID string) (string, error) {
	// Check cache first
	cacheMu.RLock()
	if size, ok := tickSizeCache[tokenID]; ok {
		cacheMu.RUnlock()
		return size, nil
	}
	cacheMu.RUnlock()

	// Fetch from API
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointTickSize)
	params := map[string]string{"token_id": tokenID}

	var result struct {
		MinimumTickSize float64 `json:"minimum_tick_size"`
	}

	if err := c.get(ctx, url, params, &result, false); err != nil {
		return "", fmt.Errorf("failed to get tick size for token %s: %w", tokenID, err)
	}

	tickSize := strconv.FormatFloat(result.MinimumTickSize, 'f', -1, 64)

	// Cache the result
	cacheMu.Lock()
	tickSizeCache[tokenID] = tickSize
	cacheMu.Unlock()

	return tickSize, nil
}

// GetNegRisk checks if a token uses negative risk (with caching)
func (c *ClobClient) GetNegRisk(ctx context.Context, tokenID string) (bool, error) {
	// Check cache first
	cacheMu.RLock()
	if negRisk, ok := negRiskCache[tokenID]; ok {
		cacheMu.RUnlock()
		return negRisk, nil
	}
	cacheMu.RUnlock()

	// Fetch from API
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointNegRisk)
	params := map[string]string{"token_id": tokenID}

	var result struct {
		NegRisk bool `json:"neg_risk"`
	}

	if err := c.get(ctx, url, params, &result, false); err != nil {
		return false, fmt.Errorf("failed to get neg risk for token %s: %w", tokenID, err)
	}

	// Cache the result
	cacheMu.Lock()
	negRiskCache[tokenID] = result.NegRisk
	cacheMu.Unlock()

	return result.NegRisk, nil
}

// GetFeeRateBps fetches the fee rate in basis points for a token (with caching)
func (c *ClobClient) GetFeeRateBps(ctx context.Context, tokenID string) (int, error) {
	// Check cache first
	cacheMu.RLock()
	if feeRate, ok := feeRateCache[tokenID]; ok {
		cacheMu.RUnlock()
		return feeRate, nil
	}
	cacheMu.RUnlock()

	// Fetch from API
	url := fmt.Sprintf("%s%s", c.baseURL, EndpointFeeRate)
	params := map[string]string{"token_id": tokenID}

	var result struct {
		BaseFee int `json:"base_fee"`
	}

	if err := c.get(ctx, url, params, &result, false); err != nil {
		return 0, fmt.Errorf("failed to get fee rate for token %s: %w", tokenID, err)
	}

	// Cache the result
	cacheMu.Lock()
	feeRateCache[tokenID] = result.BaseFee
	cacheMu.Unlock()

	return result.BaseFee, nil
}

// ClearMetadataCache clears all cached metadata (useful for testing or forcing refresh)
func ClearMetadataCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	tickSizeCache = make(map[string]string)
	negRiskCache = make(map[string]bool)
	feeRateCache = make(map[string]int)
}
