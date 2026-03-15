package clob

import (
	"context"
	"fmt"
)

// Rewards endpoints
const (
	EndpointUserEarnings              = "/rewards/user"
	EndpointTotalUserEarnings         = "/rewards/user/total"
	EndpointLiquidityRewardPercentages = "/rewards/user/percentages"
	EndpointRewardsMarketsCurrent     = "/rewards/markets/current"
	EndpointRewardsMarkets            = "/rewards/markets/"
	EndpointRewardsEarningsPercentages = "/rewards/user/markets"
)

// GetUserEarnings fetches user earnings for a specific day
func (c *ClobClient) GetUserEarnings(ctx context.Context, day string) (*UserEarning, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointUserEarnings)
	params := map[string]string{"day": day}
	
	var earning UserEarning
	if err := c.get(ctx, url, params, &earning, true); err != nil {
		return nil, fmt.Errorf("failed to get user earnings: %w", err)
	}

	return &earning, nil
}

// GetTotalUserEarnings fetches total user earnings for a specific day
func (c *ClobClient) GetTotalUserEarnings(ctx context.Context, day string) (*TotalUserEarning, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointTotalUserEarnings)
	params := map[string]string{"day": day}
	
	var earning TotalUserEarning
	if err := c.get(ctx, url, params, &earning, true); err != nil {
		return nil, fmt.Errorf("failed to get total user earnings: %w", err)
	}

	return &earning, nil
}

// GetLiquidityRewardPercentages fetches reward percentages for a market
func (c *ClobClient) GetLiquidityRewardPercentages(ctx context.Context, market string) (*RewardsPercentages, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointLiquidityRewardPercentages)
	params := map[string]string{"market": market}
	
	var percentages RewardsPercentages
	if err := c.get(ctx, url, params, &percentages, true); err != nil {
		return nil, fmt.Errorf("failed to get liquidity reward percentages: %w", err)
	}

	return &percentages, nil
}

// GetRewardsMarketsCurrentScoring fetches currently scoring markets
func (c *ClobClient) GetRewardsMarketsCurrentScoring(ctx context.Context) ([]*MarketReward, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointRewardsMarketsCurrent)
	
	var markets []*MarketReward
	if err := c.get(ctx, url, nil, &markets, false); err != nil {
		return nil, fmt.Errorf("failed to get current scoring markets: %w", err)
	}

	return markets, nil
}

// GetRewardsMarketsScoring fetches scoring info for a specific market and day
func (c *ClobClient) GetRewardsMarketsScoring(ctx context.Context, market string, day string) (*MarketReward, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s%s", c.baseURL, EndpointRewardsMarkets, market)
	params := map[string]string{"day": day}
	
	var reward MarketReward
	if err := c.get(ctx, url, params, &reward, false); err != nil {
		return nil, fmt.Errorf("failed to get market rewards scoring: %w", err)
	}

	return &reward, nil
}

// GetRewardsEarningsPercentages fetches earnings percentages breakdown
func (c *ClobClient) GetRewardsEarningsPercentages(ctx context.Context, market string, day string) (*UserRewardsEarning, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointRewardsEarningsPercentages)
	params := map[string]string{
		"market": market,
		"day":    day,
	}
	
	var earning UserRewardsEarning
	if err := c.get(ctx, url, params, &earning, true); err != nil {
		return nil, fmt.Errorf("failed to get rewards earnings percentages: %w", err)
	}

	return &earning, nil
}
