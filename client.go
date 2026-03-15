package polymarket

import (
	"context"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/QuantProcessing/polymarket-go/clients/clob"
	"github.com/QuantProcessing/polymarket-go/clients/ctf"
	"github.com/QuantProcessing/polymarket-go/clients/data"
	"github.com/QuantProcessing/polymarket-go/clients/gamma"
	"github.com/QuantProcessing/polymarket-go/clients/subgraph"

	"github.com/QuantProcessing/polymarket-go/services/market"
	"github.com/QuantProcessing/polymarket-go/services/ordermanager"
	"github.com/QuantProcessing/polymarket-go/services/realtime"
	"github.com/QuantProcessing/polymarket-go/services/trading"
)

// Config holds all configuration for the SDK
type Config struct {
	// Auth Credentials
	APIKey        string
	APISecret     string
	APIPassphrase string
	PrivateKey    string // EOA Private Key (Hex)
	FunderAddress string // Optional: Safe Address

	// Service Endpoints
	CLOBURL     string
	DataAPIURL  string
	GammaURL    string
	SubgraphURL string
	RPCURL      string
	ChainID     int64
}

// Client is the root entry point for the Polymarket SDK
type Client struct {
	// Services
	Trading      *trading.TradingService
	Market       *market.MarketService
	Realtime     *realtime.RealtimeService
	OrderManager *ordermanager.OrderManagerImpl

	// Underlying Clients (Exposed for advanced usage)
	Clob     *clob.ClobClient
	Ctf      *ctf.CTFClient
	Data     *data.DataClient
	Gamma    *gamma.GammaClient
	Subgraph *subgraph.SubgraphClient
}

// NewClient creates and initializes a new Polymarket SDK Client
func NewClient(ctx context.Context, config Config, logger *zap.SugaredLogger) (*Client, error) {
	// 1. Initialize Low-Level Clients

	// CLOB Client
	clobClient := clob.NewClient(config.CLOBURL, logger)
	creds := &clob.Credentials{
		APIKey:        config.APIKey,
		APISecret:     config.APISecret,
		APIPassphrase: config.APIPassphrase,
		PrivateKey:    config.PrivateKey,
		FunderAddress: config.FunderAddress,
		ChainID:       config.ChainID,
	}
	clobClient.WithCredentials(creds)

	// CTF Client (On-Chain)
	var ctfClient *ctf.CTFClient
	if config.RPCURL != "" {
		ctfConfig := ctf.CTFClientConfig{
			PrivateKeyHex: config.PrivateKey,
			RPCURL:        config.RPCURL,
			ChainID:       config.ChainID,
			FunderAddress: config.FunderAddress,
		}
		var err error
		ctfClient, err = ctf.NewCTFClient(ctfConfig)
		if err != nil {
			return nil, err
		}
	} else {
		log.Println("WARN:", "RPCURL not provided, CTF/On-Chain functionality will be disabled")
	}

	// Data Client
	dataClient := data.NewDataClient(&http.Client{}) // Default timeout handling inside

	// Gamma Client
	gammaClient := gamma.NewGammaClient(&http.Client{})

	// Subgraph Client
	subgraphClient := subgraph.NewSubgraphClient(&http.Client{})

	// 2. Initialize Services
	tradingService := trading.NewTradingService(clobClient, ctfClient)
	marketService := market.NewMarketService(gammaClient, dataClient, subgraphClient)
	realtimeService := realtime.NewRealtimeService(ctx)

	orderManager := ordermanager.NewOrderManager(tradingService, marketService, realtimeService, logger)

	return &Client{
		Trading:      tradingService,
		Market:       marketService,
		Realtime:     realtimeService,
		OrderManager: orderManager,
		Clob:         clobClient,
		Ctf:          ctfClient,
		Data:         dataClient,
		Gamma:        gammaClient,
		Subgraph:     subgraphClient,
	}, nil
}
