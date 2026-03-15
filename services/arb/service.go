package arb

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/QuantProcessing/polymarket-go/clients/ctf"
	"github.com/QuantProcessing/polymarket-go/clients/rtds"
	"github.com/QuantProcessing/polymarket-go/services/trading"
)

// ArbitrageService coordinates monitoring and execution of prediction market arbitrage.
//
// It monitors YES/NO token orderbooks via RTDS and detects opportunities where:
//   - Long arb: Buy YES + Buy NO costs < 1 USDC (profit on merge)
//   - Short arb: Sell YES + Sell NO revenue > 1 USDC (profit on pre-held tokens)
type ArbitrageService struct {
	config  Config
	market  *MarketConfig
	trading *trading.TradingService
	ctf     *ctf.CTFClient
	rtds    *rtds.RTDSClient
	logger  *zap.SugaredLogger

	// Lifecycle context for controlling background goroutines
	ctx    context.Context
	cancel context.CancelFunc

	// State
	mu                sync.RWMutex
	orderbook         OrderbookState
	balance           BalanceState
	isExecuting       bool
	isRunning         bool
	stats             Stats
	lastRebalanceTime time.Time

	// Channels
	stopChan chan struct{}
	doneChan chan struct{} // closed when monitorLoop exits

	// Optional recorder
	recorder *OrderbookRecorder
}

// NewArbitrageService creates a new service instance.
// The ctx parameter is the lifecycle context: when cancelled, all background goroutines will exit.
func NewArbitrageService(ctx context.Context, cfg Config, trading *trading.TradingService, ctfClient *ctf.CTFClient, rtds *rtds.RTDSClient, logger *zap.SugaredLogger) *ArbitrageService {
	childCtx, cancel := context.WithCancel(ctx)
	return &ArbitrageService{
		config:   cfg,
		trading:  trading,
		ctf:      ctfClient,
		rtds:     rtds,
		logger:   logger,
		ctx:      childCtx,
		cancel:   cancel,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// WithRecorder sets the optional orderbook CSV recorder.
func (s *ArbitrageService) WithRecorder(r *OrderbookRecorder) *ArbitrageService {
	s.recorder = r
	return s
}

// Start begins monitoring the market for arbitrage opportunities.
func (s *ArbitrageService) Start(market MarketConfig) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.market = &market
	s.isRunning = true
	s.stats.StartTime = time.Now()
	s.mu.Unlock()

	s.logger.Infow("Starting Arbitrage Service", "market", market.Name, "profitThresh", s.config.MinProfitThreshold)

	// Start recorder if configured
	if s.recorder != nil {
		if err := s.recorder.StartMarket(market.Name); err != nil {
			s.logger.Warnw("Failed to start recorder", "error", err)
		}
	}

	// Subscribe to Market Data
	err := s.rtds.SubscribeMarket([]string{market.YESTokenID, market.NOTokenID})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Initial Balance Update
	go s.UpdateBalance(s.ctx)

	// Start Loop
	go s.monitorLoop()

	return nil
}

// Stop halts the service and waits for monitorLoop to fully exit.
// This prevents goroutine races when a new service is started on the same RTDS channel.
func (s *ArbitrageService) Stop() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	close(s.stopChan)
	s.cancel()
	s.mu.Unlock()

	// Wait for monitorLoop to fully exit so it stops consuming from Messages()
	select {
	case <-s.doneChan:
	case <-time.After(2 * time.Second):
		s.logger.Warn("monitorLoop did not exit within timeout")
	}

	// Stop recorder
	if s.recorder != nil {
		s.recorder.StopMarket()
	}

	s.logger.Info("Arbitrage Service Stopped")
}

func (s *ArbitrageService) monitorLoop() {
	defer close(s.doneChan)
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.ctx.Done():
			return
		case msg := <-s.rtds.Messages():
			s.handleMessage(msg)
		}
	}
}

func (s *ArbitrageService) handleMessage(msg []byte) {
	// Skip non-JSON messages (e.g. RTDS text confirmations like "Identified")
	if len(msg) == 0 || (msg[0] != '[' && msg[0] != '{') {
		return
	}

	// Parse RTDS message into wsEvent(s)
	var events []wsEvent

	if msg[0] == '[' {
		if err := json.Unmarshal(msg, &events); err != nil {
			s.logger.Debugw("Failed to unmarshal RTDS array message", "error", err)
			return
		}
	} else {
		var single wsEvent
		if err := json.Unmarshal(msg, &single); err != nil {
			s.logger.Debugw("Failed to unmarshal RTDS message", "error", err)
			return
		}
		events = append(events, single)
	}

	updated := false
	s.mu.Lock()
	for _, e := range events {
		if s.market == nil {
			continue
		}
		switch e.AssetID {
		case s.market.YESTokenID:
			if len(e.Bids) > 0 {
				s.orderbook.YESBids = toPriceLevels(e.Bids, true)
			}
			if len(e.Asks) > 0 {
				s.orderbook.YESAsks = toPriceLevels(e.Asks, false)
			}
			s.orderbook.LastUpdate = time.Now()
			updated = true
		case s.market.NOTokenID:
			if len(e.Bids) > 0 {
				s.orderbook.NOBids = toPriceLevels(e.Bids, true)
			}
			if len(e.Asks) > 0 {
				s.orderbook.NOAsks = toPriceLevels(e.Asks, false)
			}
			s.orderbook.LastUpdate = time.Now()
			updated = true
		}
	}
	s.mu.Unlock()

	if updated {
		// Log best prices for observability
		s.mu.RLock()
		// var yesBid, yesAsk, noBid, noAsk float64
		// if len(s.orderbook.YESBids) > 0 {
		// 	yesBid = s.orderbook.YESBids[0].Price
		// }
		// if len(s.orderbook.YESAsks) > 0 {
		// 	yesAsk = s.orderbook.YESAsks[0].Price
		// }
		// if len(s.orderbook.NOBids) > 0 {
		// 	noBid = s.orderbook.NOBids[0].Price
		// }
		// if len(s.orderbook.NOAsks) > 0 {
		// 	noAsk = s.orderbook.NOAsks[0].Price
		// }
		// s.mu.RUnlock()
		// s.logger.Infow("Orderbook updated",
		// 	"yesBid", yesBid, "yesAsk", yesAsk,
		// 	"noBid", noBid, "noAsk", noAsk,
		// )

		// Record to CSV if recorder is active
		if s.recorder != nil {
			s.mu.RLock()
			ob := s.orderbook // copy
			s.mu.RUnlock()
			s.recorder.Record(&ob)
		}

		// Verify Opportunity
		opp := s.CheckOpportunity()
		if opp != nil {
			s.logger.Infow("Arbitrage Opportunity Detected", "type", opp.Type, "profit", opp.ProfitPercent)

			if s.config.AutoExecute {
				go func(o *Opportunity) {
					res, err := s.Execute(s.ctx, o)
					if err != nil {
						s.logger.Errorw("Execution failed", "type", o.Type, "error", err)
					} else if res != nil && res.Success {
						s.logger.Infow("Execution successful", "type", o.Type, "profit", res.Profit, "tx", res.TxHashes)
					}
				}(opp)
			}
		}
	}
}

// UpdateBalance fetches current USDC and token position balances from on-chain.
func (s *ArbitrageService) UpdateBalance(ctx context.Context) {
	if s.ctf == nil || s.market == nil {
		return
	}

	// Fetch USDC balance and position balances in parallel
	type usdcResult struct {
		val float64
		err error
	}
	type posResult struct {
		yes float64
		no  float64
		err error
	}

	usdcChan := make(chan usdcResult, 1)
	posChan := make(chan posResult, 1)

	go func() {
		bal, err := s.ctf.GetUsdcBalance(ctx)
		if err != nil {
			usdcChan <- usdcResult{err: err}
			return
		}
		usdcChan <- usdcResult{val: bal.InexactFloat64()}
	}()

	go func() {
		positions, err := s.ctf.GetPositionBalanceByTokenIds(ctx, s.market.ConditionID, ctf.TokenIds{
			YesTokenID: s.market.YESTokenID,
			NoTokenID:  s.market.NOTokenID,
		})
		if err != nil {
			posChan <- posResult{err: err}
			return
		}
		posChan <- posResult{
			yes: positions.YesBalance.InexactFloat64(),
			no:  positions.NoBalance.InexactFloat64(),
		}
	}()

	uRes := <-usdcChan
	pRes := <-posChan

	if uRes.err != nil {
		s.logger.Errorw("Failed to fetch USDC balance", "error", uRes.err)
		return
	}
	if pRes.err != nil {
		s.logger.Errorw("Failed to fetch position balances", "error", pRes.err)
		return
	}

	s.mu.Lock()
	s.balance.USDC = uRes.val
	s.balance.YESTokens = pRes.yes
	s.balance.NOTokens = pRes.no
	s.balance.LastUpdate = time.Now()
	s.mu.Unlock()

	s.logger.Debugw("Balance updated",
		"usdc", uRes.val,
		"yesTokens", pRes.yes,
		"noTokens", pRes.no,
	)
}
