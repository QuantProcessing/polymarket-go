package ordermanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
	"github.com/QuantProcessing/polymarket-go/services/market"
	"github.com/QuantProcessing/polymarket-go/services/realtime"
	"github.com/QuantProcessing/polymarket-go/services/trading"
)

// trackedOrder holds internal state for a monitored order
type trackedOrder struct {
	Handle  *OrderHandle
	Updates chan OrderUpdate // Writeable channel
	Status  OrderStatus
}

// OrderManagerImpl implements the OrderManager interface
type OrderManagerImpl struct {
	trading  *trading.TradingService
	market   *market.MarketService
	realtime *realtime.RealtimeService
	logger   *zap.SugaredLogger

	// Watched order state
	watchedOrders map[string]*trackedOrder
	mu            sync.RWMutex

	// Config
	pollingInterval time.Duration
}

// NewOrderManager creates a new OrderManager instance
func NewOrderManager(
	trading *trading.TradingService,
	market *market.MarketService,
	realtime *realtime.RealtimeService,
	logger *zap.SugaredLogger,
) *OrderManagerImpl {
	om := &OrderManagerImpl{
		trading:         trading,
		market:          market,
		realtime:        realtime,
		logger:          logger,
		watchedOrders:   make(map[string]*trackedOrder),
		pollingInterval: 5 * time.Second,
	}

	// Start background monitoring (Websocket + Polling fallback)
	go om.monitorLoop()

	return om
}

// CreateOrder validates and submits an order
func (om *OrderManagerImpl) CreateOrder(ctx context.Context, req OrderRequest) (*OrderResult, error) {
	// 1. Validate Core Params
	if req.Size <= 0 || req.Price <= 0 || req.TokenID == "" {
		return nil, fmt.Errorf("%w: invalid price/size/tokenID", ErrOrderValidation)
	}

	// 2. Validate Market & Balance (Optional but recommended)
	// For MVP, we skip deep validation to keep it fast, relying on CLOB response.
	// In production, we'd check `om.market.GetMarketByTokenID` and `om.trading.GetBalance`.

	// 3. Construct CLOB Request
	clobReq := clob.UserOrderParams{
		TokenID: req.TokenID,
		Price:   req.Price,
		Size:    req.Size,
		Side:    req.Side,
	}

	// 4. Submit Order
	// TradingService.CreateOrder(ctx, params, tickSize, negRisk)
	// We assume default tickSize="0.01" and negRisk=false for now (or fetch from req metadata if available)
	// TODO: Fetch real tickSize from market metadata
	tickSize := "0.01"

	resp, err := om.trading.CreateOrder(ctx, clobReq, tickSize, req.NegRisk)
	if err != nil {
		return nil, fmt.Errorf("failed to submit order: %w", err)
	}

	// 5. Create Handle
	updatesCh := make(chan OrderUpdate, 100) // Buffer updates
	handle := &OrderHandle{
		OrderID: resp.OrderID, // Assuming response has OrderID
		Request: req,
		Updates: updatesCh, // Expose as read-only
	}
	ctxHandle, cancel := context.WithCancel(context.Background())
	handle.ctx = ctxHandle
	handle.cancelFn = cancel

	// 6. Register Watcher
	tracker := &trackedOrder{
		Handle:  handle,
		Updates: updatesCh,
		Status:  StatusCreated,
	}

	om.mu.Lock()
	om.watchedOrders[resp.OrderID] = tracker
	om.mu.Unlock()

	om.logger.Infow("Order created and monitored", "orderID", resp.OrderID, "tokenID", req.TokenID)

	return &OrderResult{
		OrderID: resp.OrderID,
		Status:  StatusCreated,
		Handle:  handle,
	}, nil
}

// CancelOrder cancels an order
func (om *OrderManagerImpl) CancelOrder(ctx context.Context, orderID string) error {
	// 1. Submit Cancel Request
	err := om.trading.CancelOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Update Status locally (Optimistic)
	om.mu.RLock()
	_, exists := om.watchedOrders[orderID]
	om.mu.RUnlock()

	if exists {
		// Signal cancellation attempt?
		// Real update comes from WS, but we can emit "Cancelling" state if needed.
	}

	return nil
}

// CancelAllOrders cancels all tracked orders
func (om *OrderManagerImpl) CancelAllOrders(ctx context.Context) error {
	om.mu.Lock()
	ids := make([]string, 0, len(om.watchedOrders))
	for id := range om.watchedOrders {
		ids = append(ids, id)
	}
	om.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	return om.trading.CancelAll(ctx) // Leverages batch cancel
}

// monitorLoop listens to RealtimeService messages and polls for updates
func (om *OrderManagerImpl) monitorLoop() {
	// Start Polling Ticker
	ticker := time.NewTicker(om.pollingInterval)
	defer ticker.Stop()

	// Goroutine for WS Messages
	go func() {
		if om.realtime != nil && om.realtime.User != nil {
			for msg := range om.realtime.User.Messages() {
				om.handleUserMessage(msg)
			}
		}
	}()

	// Main Polling Loop
	for {
		select {
		case <-ticker.C:
			om.pollOrders()
			// We could also listen to a context done channel here if we added one to OrderManager
		}
	}
}

// pollOrders checks the status of all watched orders via REST API
func (om *OrderManagerImpl) pollOrders() {
	om.mu.RLock()
	ids := make([]string, 0, len(om.watchedOrders))
	for id := range om.watchedOrders {
		ids = append(ids, id)
	}
	om.mu.RUnlock()

	if len(ids) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, id := range ids {
		resp, err := om.trading.GetOrder(ctx, id)
		if err != nil {
			om.logger.Debugw("Failed to poll order", "id", id, "error", err)
			continue
		}

		om.mu.Lock()
		tracker, exists := om.watchedOrders[id]
		om.mu.Unlock()

		if !exists {
			continue
		}

		// Map CLOB status to OrderManager status
		// CLOB Statuses: "LIVE", "MATCHED", "CANCELLED", "FILLED" (varies by endpoint)
		// We treat "LIVE" / "OPEN" as StatusOpen
		// "MATCHED" / "FILLED" as StatusFilled
		// "CANCELLED" as StatusCancelled
		var newStatus OrderStatus
		s := string(resp.Status) // Convert to string for comparison

		switch s {
		case "LIVE", "OPEN":
			newStatus = StatusOpen
		case "MATCHED", "FILLED":
			newStatus = StatusFilled
		case "CANCELLED":
			newStatus = StatusCancelled
		case "EXPIRED":
			newStatus = StatusExpired
		default:
			newStatus = StatusOpen // Default to open if unknown
		}

		// Update if changed
		if newStatus != tracker.Status {
			tracker.Status = newStatus
			update := OrderUpdate{
				Status:    newStatus,
				Timestamp: time.Now(),
			}

			// Non-blocking send
			select {
			case tracker.Updates <- update:
			default:
			}

			// Terminal State Handling
			if newStatus == StatusCancelled || newStatus == StatusFilled || newStatus == StatusExpired {
				om.mu.Lock()
				if t, ok := om.watchedOrders[id]; ok {
					close(t.Updates)
					delete(om.watchedOrders, id)
				}
				om.mu.Unlock()
			}
		}
	}
}

// handleUserMessage parses raw WS messages and updates order handles
// UserEvent represents a raw event from the User WebSocket channel
type UserEvent struct {
	Type        string      `json:"type"` // "order" or "trade"?
	ID          string      `json:"id"`
	OrderID     string      `json:"order_id"`   // sometimes used in trades
	EventType   string      `json:"event_type"` // PLACEMENT, CANCELLATION, UPDATE
	Status      string      `json:"status"`     // sometimes present
	Price       interface{} `json:"price"`      // string or float
	Size        interface{} `json:"original_size"`
	SizeMatched interface{} `json:"size_matched"`
	Side        string      `json:"side"`
}

func (om *OrderManagerImpl) handleUserMessage(msg []byte) {
	// Polymarket sends arrays of events
	var events []UserEvent
	if err := json.Unmarshal(msg, &events); err != nil {
		// Try single object fallback
		var single UserEvent
		if err2 := json.Unmarshal(msg, &single); err2 == nil {
			events = append(events, single)
		} else {
			// It might be a different message type (e.g. "te" trade event), log debug
			om.logger.Debugw("Failed to unmarshal user message", "error", err, "msg", string(msg))
			return
		}
	}

	for _, event := range events {
		// Identify Order ID
		orderID := event.ID
		if orderID == "" {
			orderID = event.OrderID
		}
		if orderID == "" {
			continue
		}

		om.mu.Lock()
		state, exists := om.watchedOrders[orderID]
		om.mu.Unlock()

		if !exists {
			continue
		}

		// Map generic event to OrderStatus
		status := state.Status
		switch event.EventType {
		case "PLACEMENT":
			status = StatusOpen
		case "CANCELLATION":
			status = StatusCancelled
		case "UPDATE":
			// Check if filled
			// logic: if sizeMatched == originalSize -> Filled
			// But for now, just treat as Open/PartiallyFilled
			status = StatusOpen
		}

		// If "trade" event (fill) or matched size indicated fill
		if event.Type == "trade" {
			// Likely a fill
			// We can check event.Status if available
			status = StatusFilled // Simplifying for now, real logic needs to check amounts
		}

		newState := OrderUpdate{
			Status:    status,
			Timestamp: time.Now(),
		}

		// Update internal state
		state.Status = status

		// Unblocking send
		select {
		case state.Updates <- newState:
		default:
		}

		// Terminal State Handling
		if status == StatusCancelled || status == StatusFilled {
			// Close channel to signal completion
			om.mu.Lock()
			// Double check existence to avoid double close
			if s, ok := om.watchedOrders[orderID]; ok {
				close(s.Updates)
				delete(om.watchedOrders, orderID)
			}
			om.mu.Unlock()
		}
	}
}

/*
NOTE: The exact schema for User Events via `wss://ws-subscriptions-clob.polymarket.com/ws/user`
needs to be confirmed. Is it the same as standard CLOB WS?
If so, it often emits arrays of updates.
*/
