package arb

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
	"github.com/QuantProcessing/polymarket-go/clients/ctf"
)

// Execute handles the arbitrage opportunity
func (s *ArbitrageService) Execute(ctx context.Context, opp *Opportunity) (*ExecutionResult, error) {
	s.mu.Lock()
	if s.isExecuting {
		s.mu.Unlock()
		return nil, fmt.Errorf("another execution in progress")
	}
	s.isExecuting = true
	s.stats.ExecutionsAttempted++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isExecuting = false
		s.mu.Unlock()
		// Update balance after execution attempt
		go s.UpdateBalance(s.ctx)
	}()

	start := time.Now()

	var result *ExecutionResult
	var err error

	switch opp.Type {
	case OpportunityLong:
		result, err = s.executeLongArb(ctx, opp)
	case OpportunityShort:
		result, err = s.executeShortArb(ctx, opp)
	default:
		return nil, fmt.Errorf("unknown opportunity type: %s", opp.Type)
	}

	if err != nil {
		return &ExecutionResult{
			Success:       false,
			Type:          opp.Type,
			Error:         err,
			ExecutionTime: time.Since(start),
		}, err
	}

	result.ExecutionTime = time.Since(start)
	if result.Success {
		s.mu.Lock()
		s.stats.ExecutionsSucceeded++
		s.stats.TotalProfit += result.Profit
		s.mu.Unlock()
	}

	return result, nil
}

func (s *ArbitrageService) executeLongArb(ctx context.Context, opp *Opportunity) (*ExecutionResult, error) {
	size := opp.RecommendedSize
	market := s.market

	// Check Balance
	requiredUSDC := (opp.EffectivePrices.BuyYES + opp.EffectivePrices.BuyNO) * size
	if s.balance.USDC < requiredUSDC {
		return nil, fmt.Errorf("insufficient USDC: have %.2f, need %.2f", s.balance.USDC, requiredUSDC)
	}

	// 1. Buy tokens in parallel
	type buyResult struct {
		resp *clob.OrderResponse
		err  error
	}

	yesChan := make(chan buyResult, 1)
	noChan := make(chan buyResult, 1)

	go func() {
		// Amount is the number of outcome tokens to buy
		resp, err := s.trading.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
			TokenID: market.YESTokenID,
			Side:    clob.SideBuy,
			Amount:  size,
		}, "", false)
		yesChan <- buyResult{resp, err}
	}()

	go func() {
		resp, err := s.trading.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
			TokenID: market.NOTokenID,
			Side:    clob.SideBuy,
			Amount:  size,
		}, "", false)
		noChan <- buyResult{resp, err}
	}()

	yesRes := <-yesChan
	noRes := <-noChan

	// Check Failures
	if yesRes.err != nil || noRes.err != nil {
		// TODO: Handle partial fills — if one leg succeeded and the other failed,
		// we need to either retry the failed leg or unwind the successful one.
		return nil, fmt.Errorf("buy failed: yesErr=%v, noErr=%v", yesRes.err, noRes.err)
	}

	// 2. Merge conditional tokens back to USDC
	// Query actual on-chain position balances to determine how many tokens we can merge.
	// Market orders may partially fill, so we can't assume we got the full requested size.
	positions, err := s.ctf.GetPositionBalanceByTokenIds(ctx, market.ConditionID, ctf.TokenIds{
		YesTokenID: market.YESTokenID,
		NoTokenID:  market.NOTokenID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query position balances for merge: %w", err)
	}

	yesFilled := positions.YesBalance.InexactFloat64()
	noFilled := positions.NoBalance.InexactFloat64()
	mergeAmount := math.Min(yesFilled, noFilled)

	if mergeAmount <= 0 {
		return nil, fmt.Errorf("no tokens available to merge: yes=%.6f, no=%.6f", yesFilled, noFilled)
	}

	s.logger.Infow("Merging tokens", "yesBalance", yesFilled, "noBalance", noFilled, "mergeAmount", mergeAmount)

	txHash, err := s.ctf.Merge(ctx, market.ConditionID, fmt.Sprintf("%f", mergeAmount))
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	return &ExecutionResult{
		Success:  true,
		Type:     OpportunityLong,
		Size:     size,
		Profit:   opp.EstimatedProfit,
		TxHashes: []string{txHash},
	}, nil
}

func (s *ArbitrageService) executeShortArb(ctx context.Context, opp *Opportunity) (*ExecutionResult, error) {
	size := opp.RecommendedSize
	market := s.market

	// Check Holdings
	held := math.Min(s.balance.YESTokens, s.balance.NOTokens)
	if held < size {
		return nil, fmt.Errorf("insufficient tokens: have %.2f, need %.2f", held, size)
	}

	// Sell in parallel
	type sellResult struct {
		resp *clob.OrderResponse
		err  error
	}
	yesChan := make(chan sellResult, 1)
	noChan := make(chan sellResult, 1)

	go func() {
		resp, err := s.trading.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
			TokenID: market.YESTokenID,
			Side:    clob.SideSell,
			Amount:  size,
		}, "", false)
		yesChan <- sellResult{resp, err}
	}()

	go func() {
		resp, err := s.trading.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
			TokenID: market.NOTokenID,
			Side:    clob.SideSell,
			Amount:  size,
		}, "", false)
		noChan <- sellResult{resp, err}
	}()

	yesRes := <-yesChan
	noRes := <-noChan

	if yesRes.err != nil || noRes.err != nil {
		return nil, fmt.Errorf("sell failed: yesErr=%v, noErr=%v", yesRes.err, noRes.err)
	}

	return &ExecutionResult{
		Success: true,
		Type:    OpportunityShort,
		Size:    size,
		Profit:  opp.EstimatedProfit,
	}, nil
}
