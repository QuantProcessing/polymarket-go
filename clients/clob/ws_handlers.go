package clob

// WebSocket message structures for different channel types

// WsMarketUpdate represents market data updates
type WsMarketUpdate struct {
	EventType string `json:"event_type"`
	Market    string `json:"market"`
	AssetID   string `json:"asset_id"`
	Price     string `json:"price"`
	Volume    string `json:"volume,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// WsBookUpdate represents orderbook updates
type WsBookUpdate struct {
	EventType string       `json:"event_type"`
	Market    string       `json:"market,omitempty"`
	AssetID   string       `json:"asset_id,omitempty"`
	Bids      []PriceLevel `json:"bids,omitempty"`
	Asks      []PriceLevel `json:"asks,omitempty"`
	Timestamp int64        `json:"timestamp"`
	Hash      string       `json:"hash,omitempty"`
}

// WsTradeUpdate represents trade feed updates
type WsTradeUpdate struct {
	EventType string `json:"event_type"`
	TradeID   string `json:"id"`
	Market    string `json:"market"`
	AssetID   string `json:"asset_id"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Side      string `json:"side"`
	Maker     string `json:"maker,omitempty"`
	Taker     string `json:"taker,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// WsUserUpdate represents user order updates
type WsUserUpdate struct {
	EventType   string `json:"event_type"`
	OrderID     string `json:"order_id"`
	Market      string `json:"market,omitempty"`
	AssetID     string `json:"asset_id,omitempty"`
	Status      string `json:"status"`
	Price       string `json:"price,omitempty"`
	Size        string `json:"size,omitempty"`
	SizeMatched string `json:"size_matched,omitempty"`
	Side        string `json:"side,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}
