package ordermanager

import (
	"context"
	"errors"
	"time"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
)

// OrderStatus represents the lifecycle state of an order
type OrderStatus string

const (
	StatusCreated         OrderStatus = "created"
	StatusPending         OrderStatus = "pending"
	StatusOpen            OrderStatus = "open"
	StatusPartiallyFilled OrderStatus = "partially_filled"
	StatusFilled          OrderStatus = "filled"
	StatusCancelled       OrderStatus = "cancelled"
	StatusRejected        OrderStatus = "rejected"
	StatusExpired         OrderStatus = "expired"
)

// OrderRequest represents a request to place an order
type OrderRequest struct {
	TokenID   string
	Price     float64
	Size      float64
	Side      clob.Side
	OrderType string // "GTC", "FOK", "FAK", "GTD"
	NegRisk   bool   // Whether the market is Negative Risk
	
	// Optional metadata for tracking
	StrategyID string
	Notes      string
}

// OrderResult is the immediate result of placing an order (before monitoring)
type OrderResult struct {
	OrderID   string
	Status    OrderStatus
	Error     error
	Handle    *OrderHandle
}

// OrderHandle provides a way to track the lifecycle of an order
// It uses channels to communicate updates, offering a Go-idiomatic way to "await" 
// states or react to events.
type OrderHandle struct {
	OrderID   string
	Request   OrderRequest
	
	// Channels for lifecycle events
	// Users can select on these or use the helper methods
	Updates   <-chan OrderUpdate
	
	// Internal control
	cancelFn  context.CancelFunc
	ctx       context.Context
}

// OrderUpdate represents a change in order state
type OrderUpdate struct {
	Status      OrderStatus
	FilledSize  float64 // Incremental fill size for this update
	TotalFilled float64
	AvgPrice    float64
	Timestamp   time.Time
	Reason      string
	TxHash      string // For on-chain settlement
}

// OrderManager interface
type Manager interface {
	// CreateOrder validates, submits, and starts monitoring an order
	CreateOrder(ctx context.Context, req OrderRequest) (*OrderResult, error)
	
	// CancelOrder cancels an order and stops monitoring once confirmed
	CancelOrder(ctx context.Context, orderID string) error
	
	// CancelAllOrders cancels all open orders managed by this manager
	CancelAllOrders(ctx context.Context) error
}

var (
	ErrOrderValidation = errors.New("order validation failed")
	ErrMarketClosed    = errors.New("market is closed")
	ErrInsufficientBal = errors.New("insufficient balance")
)
