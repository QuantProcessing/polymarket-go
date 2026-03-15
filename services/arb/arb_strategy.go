package arb

import (
	"math"
	"time"
)

// CheckOpportunity evaluates the current orderbook for arbitrage opportunities.
// It returns an opportunity if profitability exceeds the configured threshold.
func (s *ArbitrageService) CheckOpportunity() *Opportunity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure we have liquidity on both sides
	if len(s.orderbook.YESBids) == 0 || len(s.orderbook.YESAsks) == 0 ||
		len(s.orderbook.NOBids) == 0 || len(s.orderbook.NOAsks) == 0 {
		return nil
	}

	// Simple check: Look at best prices first
	yesBestBid := s.orderbook.YESBids[0].Price
	yesBestAsk := s.orderbook.YESAsks[0].Price
	noBestBid := s.orderbook.NOBids[0].Price
	noBestAsk := s.orderbook.NOAsks[0].Price

	effective := EffectivePrices{
		BuyYES:   yesBestAsk,
		BuyNO:    noBestAsk,
		SellYES:  yesBestBid,
		SellNO:   noBestBid,
		LongCost: yesBestAsk + noBestAsk,
		ShortRev: yesBestBid + noBestBid,
	}

	// Calculate Profitability
	longProfit := 1.0 - effective.LongCost
	shortProfit := effective.ShortRev - 1.0

	// Determine Max Executable Size based on Orderbook Depth (Top Level)
	safetyFactor := s.config.SizeSafetyFactor

	// 1. Check Long Arb (Buy YES + Buy NO -> Merge)
	if longProfit > s.config.MinProfitThreshold {
		obAppsYes := s.orderbook.YESAsks[0].Size
		obAppsNo := s.orderbook.NOAsks[0].Size
		maxOBSize := math.Min(obAppsYes, obAppsNo) * safetyFactor

		// Balance constraint: Cost = (P_yes + P_no) * Size <= USDC Balance
		maxBalSize := 0.0
		if effective.LongCost > 0 {
			maxBalSize = s.balance.USDC / effective.LongCost
		}

		// Final Sizing
		limitSize := math.Min(maxOBSize, maxBalSize)
		limitSize = math.Min(limitSize, s.config.MaxTradeSizeSDK)

		if limitSize >= s.config.MinTradeSizeSDK {
			return &Opportunity{
				Type:             OpportunityLong,
				ProfitRate:       longProfit,
				ProfitPercent:    longProfit * 100,
				EffectivePrices:  effective,
				MaxOrderbookSize: maxOBSize,
				MaxBalanceSize:   maxBalSize,
				RecommendedSize:  limitSize,
				EstimatedProfit:  longProfit * limitSize,
				Description:      "Buy YES + NO, Merge for 1 USDC",
				Timestamp:        time.Now(),
			}
		}
	}

	// 2. Check Short Arb (Sell YES + Sell NO) -> Uses pre-held tokens
	if shortProfit > s.config.MinProfitThreshold {
		obBidsYes := s.orderbook.YESBids[0].Size
		obBidsNo := s.orderbook.NOBids[0].Size
		maxOBSize := math.Min(obBidsYes, obBidsNo) * safetyFactor

		// Balance constraint: Held pairs
		heldPairs := math.Min(s.balance.YESTokens, s.balance.NOTokens)

		// Final Sizing
		limitSize := math.Min(maxOBSize, heldPairs)
		limitSize = math.Min(limitSize, s.config.MaxTradeSizeSDK)

		// Reserve check
		if limitSize >= s.config.MinTradeSizeSDK && heldPairs >= s.config.MinTokenReserve {
			return &Opportunity{
				Type:             OpportunityShort,
				ProfitRate:       shortProfit,
				ProfitPercent:    shortProfit * 100,
				EffectivePrices:  effective,
				MaxOrderbookSize: maxOBSize,
				MaxBalanceSize:   heldPairs,
				RecommendedSize:  limitSize,
				EstimatedProfit:  shortProfit * limitSize,
				Description:      "Sell YES + NO (Pre-held)",
				Timestamp:        time.Now(),
			}
		}
	}

	return nil
}

// CalculateEffectivePrices is a helper for more advanced depth calculation (Future use)
func CalculateEffectivePrices(bids, asks []PriceLevel, size float64) float64 {
	// TODO: Implement VWAP if needed for larger sizes
	if len(asks) > 0 {
		return asks[0].Price
	}
	return 0
}
