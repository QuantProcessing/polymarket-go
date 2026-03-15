package clob

import (
	"context"
	"fmt"
)

// Account endpoints
const (
	EndpointBalanceAllowance       = "/balance-allowance"
	EndpointUpdateBalanceAllowance = "/balance-allowance/update"
	EndpointClosedOnlyMode         = "/auth/ban-status/closed-only"
)

// GetBalanceAllowance fetches the USDC balance and allowance for trading
func (c *ClobClient) GetBalanceAllowance(ctx context.Context, params BalanceAllowanceParams) (*BalanceAllowanceResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointBalanceAllowance)
	
	// Add signature type to params
	queryParams := make(map[string]string)
	if params.AssetType != "" {
		queryParams["asset_type"] = params.AssetType
	}
	if params.TokenID != "" {
		queryParams["token_id"] = params.TokenID
	}
	// Note: signature_type param omitted - will use default
	
	var resp BalanceAllowanceResponse
	if err := c.get(ctx, url, queryParams, &resp, true); err != nil {
		return nil, fmt.Errorf("failed to get balance/allowance: %w", err)
	}

	return &resp, nil
}

// UpdateBalanceAllowance updates the USDC allowance for trading
func (c *ClobClient) UpdateBalanceAllowance(ctx context.Context, params BalanceAllowanceParams) error {
	if err := c.EnsureAuth(ctx); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointUpdateBalanceAllowance)
	
	// Add signature type to params
	queryParams := make(map[string]string)
	if params.AssetType != "" {
		queryParams["asset_type"] = params.AssetType
	}
	if params.TokenID != "" {
		queryParams["token_id"] = params.TokenID
	}
	if params.Signature != "" {
		queryParams["signature"] = params.Signature
	}
	// Note: signature_type param omitted - will use default
	
	if err := c.get(ctx, url, queryParams, nil, true); err != nil {
		return fmt.Errorf("failed to update balance/allowance: %w", err)
	}

	return nil
}

// GetClosedOnlyMode checks if the account is in closed-only (ban) mode
func (c *ClobClient) GetClosedOnlyMode(ctx context.Context) (*BanStatus, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointClosedOnlyMode)
	
	var status BanStatus
	if err := c.get(ctx, url, nil, &status, true); err != nil {
		return nil, fmt.Errorf("failed to get ban status: %w", err)
	}

	return &status, nil
}

// ApproveUSDC is a helper method to approve USDC spending for trading
// This typically requires the user to sign a transaction on-chain
// For now, this is a placeholder that would need integration with web3 wallet
func (c *ClobClient) ApproveUSDC(ctx context.Context, amount string) error {
	// Note: This would require web3 wallet integration to actually call
	// the USDC contract's approve() method. For now, users should handle
	// this manually through the Polymarket UI or their wallet.
	return fmt.Errorf("ApproveUSDC: not implemented - please approve USDC manually through Polymarket UI")
}
