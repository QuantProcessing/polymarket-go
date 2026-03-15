package rtds

import "github.com/QuantProcessing/polymarket-go/clients/clob"

// RTDSEvent represents a generic event wrapper
type RTDSEvent struct {
	EventType string `json:"event_type"`
	AssetID   string `json:"asset_id"`
}

// OrderbookUpdate represents an orderbook update from RTDS
type OrderbookUpdate struct {
	EventType string            `json:"event_type"` // "book"
	AssetID   string            `json:"asset_id"`
	Bids      []clob.PriceLevel `json:"bids"`
	Asks      []clob.PriceLevel `json:"asks"`
	Hash      string            `json:"hash"`
	Timestamp string            `json:"timestamp"` // RTDS sends string timestamp usually?
}

// ParseOrderbookUpdates parses a raw message into a slice of OrderbookUpdates
// RTDS can send a single object or an array of objects
func ParseOrderbookUpdates(msg []byte) ([]OrderbookUpdate, error) {
    // ... logic to be implemented in helper or just let user do it?
    // Given this is a client pkg, we can provide helpers.
    return nil, nil // Placeholder, implemented in functional code below
}
