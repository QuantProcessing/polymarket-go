package market

import (
	"context"

	"github.com/QuantProcessing/polymarket-go/clients/data"
	"github.com/QuantProcessing/polymarket-go/clients/gamma"
	"github.com/QuantProcessing/polymarket-go/clients/subgraph"
)

// MarketService aggregates market discovery and data retrieval
type MarketService struct {
	Gamma    *gamma.GammaClient
	Data     *data.DataClient
	Subgraph *subgraph.SubgraphClient
}

func NewMarketService(gammaClient *gamma.GammaClient, dataClient *data.DataClient, subgraphClient *subgraph.SubgraphClient) *MarketService {
	return &MarketService{
		Gamma:    gammaClient,
		Data:     dataClient,
		Subgraph: subgraphClient,
	}
}

// GetMarketByConditionID fetches market details by condition ID
func (s *MarketService) GetMarketByConditionID(ctx context.Context, conditionID string) (*gamma.GammaMarket, error) {
	return s.Gamma.GetMarketByConditionID(ctx, conditionID)
}

// GetMarkets searches for markets via Gamma
func (s *MarketService) GetMarkets(ctx context.Context, params gamma.MarketSearchParams) ([]gamma.GammaMarket, error) {
	return s.Gamma.GetMarkets(ctx, params)
}

// GetTrades fetches trade history via Data API
func (s *MarketService) GetTrades(params data.TradesParams) ([]data.DataTrade, error) {
	return s.Data.GetTrades(params)
}

// GetPositions fetches active positions via Data API
func (s *MarketService) GetPositions(address string, params data.PositionsParams) ([]data.Position, error) {
	return s.Data.GetPositions(address, params)
}
