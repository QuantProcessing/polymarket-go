package prediction

import (
	"time"
)

// Market represents a prediction market event.
type Market struct {
	ID          string    `json:"id"`
	Question    string    `json:"question"`
	Description string    `json:"description"`
	EndTime     time.Time `json:"end_time"`
	Outcomes    []Outcome `json:"outcomes"`
	Active      bool      `json:"active"`
	Closed      bool      `json:"closed"`
	Archived    bool      `json:"archived"`
}

// Outcome represents a possible result in a market.
type Outcome struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"` // e.g. "Yes", "No", "Trump", "Biden"
	Price       float64 `json:"price"`
	Probability float64 `json:"probability"`
}

// Token represents the tradeable asset for an outcome.
type Token struct {
	ID        string `json:"id"`
	Symbol    string `json:"symbol"` // e.g. "YES", "NO"
	TokenID   string `json:"token_id"`
	Decimals  int    `json:"decimals"`
	Address   string `json:"address"`
	OutcomeID string `json:"outcome_id"`
}

// Orderbook represents the state of the order book for a specific token/outcome.
type Orderbook struct {
	MarketID  string      `json:"market_id"`
	TokenID   string      `json:"token_id"`
	Timestamp int64       `json:"timestamp"`
	Bids      []Level `json:"bids"`
	Asks      []Level `json:"asks"`
}

// Level represents a price level in the orderbook.
type Level struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// Side is order side (buy/sell).
type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

// OrderType is order type (limit/market/etc).
type OrderType string

const (
	GTC OrderType = "GTC" // Good Till Cancelled
	IOC OrderType = "IOC" // Immediate Or Cancel
	FOK OrderType = "FOK" // Fill Or Kill
)
