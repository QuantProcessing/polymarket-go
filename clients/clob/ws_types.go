package clob

// WebSocket Event Types for User Channel

// WsOrderEvent represents an order lifecycle event (PLACEMENT, UPDATE, CANCELLATION)
type WsOrderEvent struct {
	EventType    string `json:"event_type"`    // "order"
	ID           string `json:"id"`            // Order ID
	Type         string `json:"type"`          // "PLACEMENT", "UPDATE", "CANCELLATION"
	AssetID      string `json:"asset_id"`      // Token ID
	Market       string `json:"market"`        // Condition ID
	OrderOwner   string `json:"order_owner"`   // Owner UUID
	Owner        string `json:"owner"`         // Owner address
	OriginalSize string `json:"original_size"` // Original order size
	Price        string `json:"price"`         // Order price
	Side         string `json:"side"`          // "BUY" or "SELL"
	SizeMatched  string `json:"size_matched"`  // Size that has been matched
	Outcome      string `json:"outcome"`       // "YES" or "NO"
	Timestamp    string `json:"timestamp"`     // Unix timestamp
}

// WsTradeEvent represents a trade execution event (MATCHED, MINED, CONFIRMED, FAILED)
type WsTradeEvent struct {
	EventType    string          `json:"event_type"` // "trade"
	ID           string          `json:"id"`         // Trade ID
	Type         string          `json:"type"`       // "TRADE"
	Status       string          `json:"status"`     // "MATCHED", "MINED", "CONFIRMED", "RETRYING", "FAILED"
	AssetID      string          `json:"asset_id"`   // Token ID
	Market       string          `json:"market"`     // Condition ID
	TakerOrderID string          `json:"taker_order_id"`
	MakerOrders  []WsMakerOrder  `json:"maker_orders"`
	Size         string          `json:"size"`       // Trade size
	Price        string          `json:"price"`      // Trade price
	Side         string          `json:"side"`       // "BUY" or "SELL"
	Outcome      string          `json:"outcome"`    // "YES" or "NO"
	Owner        string          `json:"owner"`      // Trade owner
	TradeOwner   string          `json:"trade_owner"`
	MatchTime    string          `json:"matchtime"`  // Match timestamp
	LastUpdate   string          `json:"last_update"`
	Timestamp    string          `json:"timestamp"`
}

// WsMakerOrder represents a maker order in a trade
type WsMakerOrder struct {
	OrderID       string `json:"order_id"`
	AssetID       string `json:"asset_id"`
	MatchedAmount string `json:"matched_amount"`
	Outcome       string `json:"outcome"`
	Owner         string `json:"owner"`
	Price         string `json:"price"`
}

// WebSocket Event Types for Market Channel

// WsBookEvent represents an orderbook update event
type WsBookEvent struct {
	EventType string          `json:"event_type"` // "book"
	AssetID   string          `json:"asset_id"`   // Token ID
	Market    string          `json:"market"`     // Condition ID
	Timestamp string          `json:"timestamp"`  // Unix timestamp
	Hash      string          `json:"hash"`       // Book hash
	Bids      []WsBookLevel   `json:"bids"`       // Bid levels [price, size]
	Asks      []WsBookLevel   `json:"asks"`       // Ask levels [price, size]
}

// WsBookLevel represents a single orderbook price level
type WsBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// WsPriceChangeEvent represents a price change event
type WsPriceChangeEvent struct {
	EventType    string          `json:"event_type"`    // "price_change"
	Market       string          `json:"market"`        // Condition ID
	PriceChanges []WsPriceChange `json:"price_changes"` // Array of price changes
	Timestamp    string          `json:"timestamp"`     // Unix timestamp
}

// WsPriceChange represents a single price change
type WsPriceChange struct {
	AssetID string `json:"asset_id"` // Token ID
	Price   string `json:"price"`    // New price
	Size    string `json:"size"`     // Trade size
	Side    string `json:"side"`     // "BUY" or "SELL"
	Hash    string `json:"hash"`     // Trade hash
	BestBid string `json:"best_bid"` // Current best bid
	BestAsk string `json:"best_ask"` // Current best ask
}

// WsTickSizeChangeEvent represents a tick size change event
type WsTickSizeChangeEvent struct {
	EventType string `json:"event_type"` // "tick_size_change"
	AssetID   string `json:"asset_id"`   // Token ID
	Market    string `json:"market"`     // Condition ID
	TickSize  string `json:"tick_size"`  // New tick size
	Timestamp string `json:"timestamp"`  // Unix timestamp
}

// WsLastTradePriceEvent represents the last trade price event
type WsLastTradePriceEvent struct {
	EventType      string `json:"event_type"`       // "last_trade_price"
	AssetID        string `json:"asset_id"`         // Token ID
	Market         string `json:"market"`           // Condition ID
	LastTradePrice string `json:"last_trade_price"` // Last trade price
	Timestamp      string `json:"timestamp"`        // Unix timestamp
}

// WsBestBidAskEvent represents best bid/ask update event
type WsBestBidAskEvent struct {
	EventType string `json:"event_type"` // "best_bid_ask"
	AssetID   string `json:"asset_id"`   // Token ID
	Market    string `json:"market"`     // Condition ID
	BestBid   string `json:"best_bid"`   // Best bid price
	BestAsk   string `json:"best_ask"`   // Best ask price
	Timestamp string `json:"timestamp"`  // Unix timestamp
}

// WsNewMarketEvent represents a new market creation event
type WsNewMarketEvent struct {
	EventType string `json:"event_type"` // "new_market"
	Market    string `json:"market"`     // Condition ID
	Question  string `json:"question"`   // Market question
	Timestamp string `json:"timestamp"`  // Unix timestamp
}

// WsMarketResolvedEvent represents a market resolution event
type WsMarketResolvedEvent struct {
	EventType string `json:"event_type"` // "market_resolved"
	Market    string `json:"market"`     // Condition ID
	Outcome   string `json:"outcome"`    // Resolution outcome
	Timestamp string `json:"timestamp"`  // Unix timestamp
}

// WsBaseEvent is a minimal structure to determine event type
type WsBaseEvent struct {
	EventType string `json:"event_type"` // "order", "trade", "book", "price_change", etc.
}
