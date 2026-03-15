package arb

import (
	"fmt"
	"math"
	"time"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
)

// RebalanceActionType defines action
type RebalanceActionType string

const (
	ActionSplit   RebalanceActionType = "split"
	ActionMerge   RebalanceActionType = "merge"
	ActionSellYES RebalanceActionType = "sell_yes"
	ActionSellNO  RebalanceActionType = "sell_no"
	ActionNone    RebalanceActionType = "none"
)

type RebalanceAction struct {
	Type     RebalanceActionType
	Amount   float64
	Reason   string
	Priority int
}

type RebalanceResult struct {
	Success bool
	Action  RebalanceAction
	TxHash  string
	Error   error
}

// CheckAndRebalance checks if rebalance is needed and executes it
func (s *ArbitrageService) CheckAndRebalance() {
	if !s.config.EnableRebalancer || !s.isRunning || s.isExecuting {
		return
	}

	// Cooldown check
	s.mu.RLock()
	lastRebalance := s.lastRebalanceTime
	s.mu.RUnlock()

	if time.Since(lastRebalance) < s.config.RebalanceCooldown {
		return
	}

	action := s.calculateRebalanceAction()

	if action.Type != ActionNone && action.Amount >= s.config.MinTradeSizeSDK {
		go s.rebalance(action)
	}
}

func (s *ArbitrageService) calculateRebalanceAction() RebalanceAction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Need valid balance + market
	if s.market == nil {
		return RebalanceAction{Type: ActionNone}
	}

	usdc := s.balance.USDC
	yes := s.balance.YESTokens
	no := s.balance.NOTokens

	paired := math.Min(yes, no)
	currentTotal := usdc + paired // Capital definition: USDC + pairs (which can be redeemed for 1 USDC)

	if currentTotal == 0 {
		return RebalanceAction{Type: ActionNone}
	}

	usdcRatio := usdc / currentTotal
	imbalance := yes - no

	// Priority 1: Risk Control (Imbalance)
	if math.Abs(imbalance) > s.config.ImbalanceThreshold {
		if imbalance > 0 {
			sellAmt := math.Min(imbalance, yes*0.5)
			if sellAmt >= s.config.MinTradeSizeSDK {
				return RebalanceAction{
					Type:     ActionSellYES,
					Amount:   math.Floor(sellAmt*1e6) / 1e6,
					Reason:   fmt.Sprintf("Risk: YES > NO by %.2f", imbalance),
					Priority: 100,
				}
			}
		} else {
			sellAmt := math.Min(-imbalance, no*0.5)
			if sellAmt >= s.config.MinTradeSizeSDK {
				return RebalanceAction{
					Type:     ActionSellNO,
					Amount:   math.Floor(sellAmt*1e6) / 1e6,
					Reason:   fmt.Sprintf("Risk: NO > YES by %.2f", -imbalance),
					Priority: 100,
				}
			}
		}
	}

	// Priority 2: High USDC -> Split
	if usdcRatio > s.config.MaxUSDCRatio {
		targetUSDC := currentTotal * s.config.TargetUSDCRatio
		excess := usdc - targetUSDC
		splitAmt := math.Min(excess*0.5, usdc*0.3)

		if splitAmt >= s.config.MinTradeSizeSDK {
			return RebalanceAction{
				Type:     ActionSplit,
				Amount:   math.Floor(splitAmt*100) / 100,
				Reason:   fmt.Sprintf("USDC %.0f%% > %.0f%% max", usdcRatio*100, s.config.MaxUSDCRatio*100),
				Priority: 50,
			}
		}
	}

	// Priority 3: Low USDC -> Merge
	if usdcRatio < s.config.MinUSDCRatio && paired >= s.config.MinTradeSizeSDK {
		targetUSDC := currentTotal * s.config.TargetUSDCRatio
		needed := targetUSDC - usdc
		mergeAmt := math.Min(needed*0.5, paired*0.5)

		if mergeAmt >= s.config.MinTradeSizeSDK {
			return RebalanceAction{
				Type:     ActionMerge,
				Amount:   math.Floor(mergeAmt*100) / 100,
				Reason:   fmt.Sprintf("USDC %.0f%% < %.0f%% min", usdcRatio*100, s.config.MinUSDCRatio*100),
				Priority: 50,
			}
		}
	}

	return RebalanceAction{Type: ActionNone}
}

func (s *ArbitrageService) rebalance(action RebalanceAction) {
	ctx := s.ctx // Use lifecycle context instead of context.Background()
	s.logger.Infow("Rebalancing", "action", action.Type, "amount", action.Amount, "reason", action.Reason)

	var err error
	var txHash string

	switch action.Type {
	case ActionSplit:
		txHash, err = s.trading.Split(ctx, s.market.ConditionID, action.Amount)
	case ActionMerge:
		txHash, err = s.trading.Merge(ctx, s.market.ConditionID, action.Amount)
	case ActionSellYES:
		_, err = s.trading.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
			TokenID: s.market.YESTokenID,
			Side:    clob.SideSell,
			Amount:  action.Amount,
		}, "", false)
	case ActionSellNO:
		_, err = s.trading.Clob.CreateMarketOrder(ctx, clob.UserMarketOrderParams{
			TokenID: s.market.NOTokenID,
			Side:    clob.SideSell,
			Amount:  action.Amount,
		}, "", false)
	}

	if err != nil {
		s.logger.Errorw("Rebalance Failed", "error", err)
	} else {
		s.logger.Infow("Rebalance Success", "tx", txHash)
		// Update cooldown + balance
		s.mu.Lock()
		s.lastRebalanceTime = time.Now()
		s.mu.Unlock()
		s.UpdateBalance(ctx)
	}
}
