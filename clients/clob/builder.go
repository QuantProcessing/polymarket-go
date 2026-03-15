package clob

import (
	"context"
	"fmt"
)

// Builder API endpoints
const (
	EndpointBuilderTrades     = "/builder/trades"
	EndpointBuilderAPIKey     = "/auth/builder-api-key"
	EndpointBuilderAPIKeys    = "/auth/builder-api-key"
	EndpointRevokeBuilderKey  = "/auth/builder-api-key"
)

// Readonly API endpoints
const (
	EndpointCreateReadonlyKey   = "/auth/readonly-api-key"
	EndpointGetReadonlyKeys     = "/auth/readonly-api-keys"
	EndpointDeleteReadonlyKey   = "/auth/readonly-api-key"
	EndpointValidateReadonlyKey = "/auth/validate-readonly-api-key"
)

// GetBuilderTrades fetches trade history for builder
func (c *ClobClient) GetBuilderTrades(ctx context.Context, params TradeParams, nextCursor string) (*PaginationPayload, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointBuilderTrades)
	
	queryParams := make(map[string]string)
	if nextCursor != "" {
		queryParams["next_cursor"] = nextCursor
	}
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
		return nil, fmt.Errorf("failed to get builder trades: %w", err)
	}

	return &result, nil
}

// CreateBuilderAPIKey creates a new builder API key
func (c *ClobClient) CreateBuilderAPIKey(ctx context.Context, builderConfig BuilderConfig) (*BuilderApiKeyResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointBuilderAPIKey)
	
	var response BuilderApiKeyResponse
	if err := c.post(ctx, url, builderConfig, &response, true); err != nil {
		return nil, fmt.Errorf("failed to create builder API key: %w", err)
	}

	return &response, nil
}
//...
// RevokeBuilderAPIKey revokes a builder API key
func (c *ClobClient) RevokeBuilderAPIKey(ctx context.Context, builderConfig BuilderConfig, key string) (bool, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return false, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointRevokeBuilderKey)
	
	payload := map[string]string{"key": key}
	
	if err := c.delete(ctx, url, payload, nil, true); err != nil {
		return false, fmt.Errorf("failed to revoke builder API key: %w", err)
	}

	return true, nil
}
//...
// DeleteReadonlyAPIKey deletes a readonly API key
func (c *ClobClient) DeleteReadonlyAPIKey(ctx context.Context, key string) (bool, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return false, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointDeleteReadonlyKey)
	
	payload := map[string]string{"key": key}
	
	if err := c.delete(ctx, url, payload, nil, true); err != nil {
		return false, fmt.Errorf("failed to delete readonly API key: %w", err)
	}

	return true, nil
}

// ValidateReadonlyAPIKey validates a readonly API key
func (c *ClobClient) ValidateReadonlyAPIKey(ctx context.Context, address string, key string) (string, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return "", fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointValidateReadonlyKey)
	
	params := map[string]string{
		"address": address,
		"key":     key,
	}
	
	var result string
	if err := c.get(ctx, url, params, &result, true); err != nil {
		return "", fmt.Errorf("failed to validate readonly API key: %w", err)
	}

	return result, nil
}
