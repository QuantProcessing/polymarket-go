package realtime

import (
	"context"

	"github.com/QuantProcessing/polymarket-go/clients/rtds"
)

// RealtimeService manages centralized WebSocket connections (Market & User channels)
type RealtimeService struct {
	Market *rtds.RTDSClient
	User   *rtds.RTDSClient
}

func NewRealtimeService(ctx context.Context) *RealtimeService {
	// Initialize Market Channel Client
	marketConfig := rtds.RTDSConfig{
		URL:           rtds.RTDSMarketURL,
		AutoReconnect: true,
	}
	marketClient := rtds.NewRTDSClient(ctx, marketConfig)

	// Initialize User Channel Client
	userConfig := rtds.RTDSConfig{
		URL:           rtds.RTDSUserURL,
		AutoReconnect: true,
	}
	userClient := rtds.NewRTDSClient(ctx, userConfig)

	return &RealtimeService{
		Market: marketClient,
		User:   userClient,
	}
}

// ConnectAll connects both market and user clients
func (s *RealtimeService) ConnectAll() error {
	if err := s.Market.Connect(); err != nil {
		return err
	}
	if err := s.User.Connect(); err != nil {
		return err
	}
	return nil
}

// CloseAll closes all connections
func (s *RealtimeService) CloseAll() {
	s.Market.Close()
	s.User.Close()
}
