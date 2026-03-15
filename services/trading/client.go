package trading

import (
	"context"
	"fmt"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
	"github.com/QuantProcessing/polymarket-go/clients/ctf"
)

// TradingService handles all trading execution (orders) and asset management (split/merge)
type TradingService struct {
	Clob *clob.ClobClient
	Ctf  *ctf.CTFClient
}

func NewTradingService(clobClient *clob.ClobClient, ctfClient *ctf.CTFClient) *TradingService {
	return &TradingService{
		Clob: clobClient,
		Ctf:  ctfClient,
	}
}

// CreateOrder places a new order via CLOB
func (s *TradingService) CreateOrder(ctx context.Context, params clob.UserOrderParams, tickSize string, negRisk bool) (*clob.OrderResponse, error) {
	return s.Clob.CreateOrder(ctx, params, tickSize, negRisk)
}

// CancelOrder cancels an order
func (s *TradingService) CancelOrder(ctx context.Context, orderID string) error {
	return s.Clob.CancelOrder(ctx, orderID)
}

// GetOrder fetches a single order status
func (s *TradingService) GetOrder(ctx context.Context, orderID string) (*clob.OrderResponse, error) {
	return s.Clob.GetOrder(ctx, orderID)
}

// CancelAll cancels all orders
func (s *TradingService) CancelAll(ctx context.Context) error {
	return s.Clob.CancelAllOrders(ctx)
}

// Split mints conditional tokens
func (s *TradingService) Split(ctx context.Context, conditionID string, amount float64) (string, error) {
	return s.Ctf.Split(ctx, conditionID, fmt.Sprintf("%f", amount))
}

// Merge redeems conditional tokens
func (s *TradingService) Merge(ctx context.Context, conditionID string, amount float64) (string, error) {
	return s.Ctf.Merge(ctx, conditionID, fmt.Sprintf("%f", amount))
}

// Redeem redeems positions on a resolved market
func (s *TradingService) Redeem(ctx context.Context, conditionID string, outcome string) (string, error) {
	return s.Ctf.Redeem(ctx, conditionID, outcome)
}
