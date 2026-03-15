package arb

import (
	"sort"
	"strconv"
	"time"
)

// MarketConfig defines the market to monitor for arbitrage
type MarketConfig struct {
	Name        string    `json:"name"`
	ConditionID string    `json:"conditionId"`
	YESTokenID  string    `json:"yesTokenId"`
	NOTokenID   string    `json:"noTokenId"`
	Outcomes    [2]string `json:"outcomes"` // ["YES", "NO"]
}

// Config holds service configuration
type Config struct {
	// Trading Config
	MinProfitThreshold float64       `json:"minProfitThreshold"` // e.g. 0.005 for 0.5%
	MinTradeSizeSDK    float64       `json:"minTradeSize"`       // in USDC
	MaxTradeSizeSDK    float64       `json:"maxTradeSize"`       // in USDC
	MinTokenReserve    float64       `json:"minTokenReserve"`    // Min tokens to keep for short arb
	AutoExecute        bool          `json:"autoExecute"`
	ExecutionCooldown  time.Duration `json:"executionCooldown"`

	// Rebalancer Config
	EnableRebalancer   bool          `json:"enableRebalancer"`
	MinUSDCRatio       float64       `json:"minUsdcRatio"`    // e.g. 0.2
	MaxUSDCRatio       float64       `json:"maxUsdcRatio"`    // e.g. 0.8
	TargetUSDCRatio    float64       `json:"targetUsdcRatio"` // e.g. 0.5
	ImbalanceThreshold float64       `json:"imbalanceThreshold"`
	RebalanceInterval  time.Duration `json:"rebalanceInterval"`
	RebalanceCooldown  time.Duration `json:"rebalanceCooldown"`
	SizeSafetyFactor   float64       `json:"sizeSafetyFactor"` // e.g. 0.8
	AutoFixImbalance   bool          `json:"autoFixImbalance"`
}

// DefaultConfig returns safe defaults
func DefaultConfig() Config {
	return Config{
		MinProfitThreshold: 0.005,
		MinTradeSizeSDK:    5.0,
		MaxTradeSizeSDK:    100.0,
		MinTokenReserve:    10.0,
		AutoExecute:        false,
		ExecutionCooldown:  5 * time.Second,

		EnableRebalancer:   false,
		MinUSDCRatio:       0.2,
		MaxUSDCRatio:       0.8,
		TargetUSDCRatio:    0.5,
		ImbalanceThreshold: 5.0,
		RebalanceInterval:  10 * time.Second,
		RebalanceCooldown:  30 * time.Second,
		SizeSafetyFactor:   0.8,
		AutoFixImbalance:   true,
	}
}

// OpportunityType defines the arb strategy
type OpportunityType string

const (
	OpportunityLong  OpportunityType = "long"  // Buy YES + NO -> Merge
	OpportunityShort OpportunityType = "short" // Sell YES + NO (pre-held)
)

// EffectivePrices holds calculated effective prices
type EffectivePrices struct {
	BuyYES   float64
	BuyNO    float64
	SellYES  float64
	SellNO   float64
	LongCost float64
	ShortRev float64
}

// Opportunity represents a detected arbitrage chance
type Opportunity struct {
	Type            OpportunityType
	ProfitRate      float64
	ProfitPercent   float64
	EffectivePrices EffectivePrices

	MaxOrderbookSize float64
	MaxBalanceSize   float64
	RecommendedSize  float64
	EstimatedProfit  float64

	Description string
	Timestamp   time.Time
}

// ExecutionResult tracks the outcome of an arb execution
type ExecutionResult struct {
	Success       bool
	Type          OpportunityType
	Size          float64
	Profit        float64
	TxHashes      []string
	Error         error
	ExecutionTime time.Duration
}

// Stats holds runtime statistics
type Stats struct {
	OpportunitiesDetected int64
	ExecutionsAttempted   int64
	ExecutionsSucceeded   int64
	TotalProfit           float64
	StartTime             time.Time
}

// BalanceState tracks current portfolio state
type BalanceState struct {
	USDC       float64
	YESTokens  float64
	NOTokens   float64
	LastUpdate time.Time
}

// OrderbookState tracks current orderbook
type OrderbookState struct {
	YESBids    []PriceLevel
	YESAsks    []PriceLevel
	NOBids     []PriceLevel
	NOAsks     []PriceLevel
	LastUpdate time.Time
}

// PriceLevel represents a single price level in the orderbook (internal)
type PriceLevel struct {
	Price float64
	Size  float64
}

// wsPriceLevel matches the RTDS JSON format where price/size are strings
type wsPriceLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// wsEvent represents a WebSocket event from RTDS
// RTDS sends price/size as strings and timestamp as ISO 8601 string
type wsEvent struct {
	Type      string         `json:"type"`
	AssetID   string         `json:"asset_id"`
	Bids      []wsPriceLevel `json:"bids"`
	Asks      []wsPriceLevel `json:"asks"`
	Timestamp string         `json:"timestamp"`
}

// toPriceLevels converts wsPriceLevels to internal PriceLevels, skipping invalid entries.
// If descending is true, sorts by price descending (for bids); otherwise ascending (for asks).
func toPriceLevels(raw []wsPriceLevel, descending bool) []PriceLevel {
	result := make([]PriceLevel, 0, len(raw))
	for _, r := range raw {
		price, err := strconv.ParseFloat(r.Price, 64)
		if err != nil {
			continue
		}
		size, err := strconv.ParseFloat(r.Size, 64)
		if err != nil {
			continue
		}
		if size == 0 {
			continue // Skip empty levels
		}
		result = append(result, PriceLevel{Price: price, Size: size})
	}
	if descending {
		sort.Slice(result, func(i, j int) bool { return result[i].Price > result[j].Price })
	} else {
		sort.Slice(result, func(i, j int) bool { return result[i].Price < result[j].Price })
	}
	return result
}
