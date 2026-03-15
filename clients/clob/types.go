package clob

// Side represents the side of an order
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeGTC OrderType = "GTC" // Good til cancelled
	OrderTypeGTD OrderType = "GTD" // Good til date
	OrderTypeFOK OrderType = "FOK" // Fill or kill
	OrderTypeFAK OrderType = "FAK" // Fill and kill
)

// Market represents a Polymarket conditional market
type Market struct {
	EnableOrderBook         bool     `json:"enable_order_book"`
	Active                  bool     `json:"active"`
	Closed                  bool     `json:"closed"`
	Archived                bool     `json:"archived"`
	AcceptingOrders         bool     `json:"accepting_orders"`
	AcceptingOrderTimestamp string   `json:"accepting_order_timestamp"`
	MinimumOrderSize        int      `json:"minimum_order_size"`
	MinimumTickSize         float64  `json:"minimum_tick_size"`
	ConditionID             string   `json:"condition_id"`
	QuestionID              string   `json:"question_id"`
	Question                string   `json:"question"`
	Description             string   `json:"description"`
	MarketSlug              string   `json:"market_slug"`
	EndDateISO              string   `json:"end_date_iso"`
	GameStartTime           string   `json:"game_start_time"`
	SecondsDelay            int      `json:"seconds_delay"`
	FPMM                    string   `json:"fpmm"`
	MakerBaseFee            int      `json:"maker_base_fee"`
	TakerBaseFee            int      `json:"taker_base_fee"`
	NotificationsEnabled    bool     `json:"notifications_enabled"`
	NegRisk                 bool     `json:"neg_risk"`
	NegRiskMarketID         string   `json:"neg_risk_market_id"`
	NegRiskRequestID        string   `json:"neg_risk_request_id"`
	Icon                    string   `json:"icon"`
	Image                   string   `json:"image"`
	Rewards                 Reward   `json:"rewards"`
	Is5050Outcome           bool     `json:"is_50_50_outcome"`
	Tokens                  []Token  `json:"tokens,omitempty"`
	Tags                    []string `json:"tags"`
}
type Reward struct {
	Rates     []Rate  `json:"rates"`
	MinSize   int     `json:"min_size"`
	MaxSpread float64 `json:"max_spread"`
}
type Rate struct {
	AssetAddress     string `json:"asset_address"`
	RewardsDailyRate int    `json:"rewards_daily_rate"`
}
type Token struct {
	TokenID string  `json:"token_id"`
	Outcome string  `json:"outcome"`
	Price   float64 `json:"price"`
	Winner  bool    `json:"winner"`
}

// PaginationPayload represents paginated API response
type PaginationPayload struct {
	NextCursor string        `json:"next_cursor"`
	Data       []interface{} `json:"data"`
}

// OrderBookSummary represents order book data
type OrderBookSummary struct {
	Market       string       `json:"market"`
	AssetID      string       `json:"asset_id"`
	Hash         string       `json:"hash,omitempty"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
	MinOrderSize string       `json:"min_order_size"`
	TickSize     string       `json:"tick_size"`
	Timestamp    string       `json:"timestamp,omitempty"`
}

// PriceLevel represents a single price level in order book
type PriceLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// BookParams for batch orderbook queries
type BookParams struct {
	TokenID string `json:"token_id"`
}

type MarketPriceHistory struct {
	History []MarketPrice `json:"history"`
}

// MarketPrice represents a price data point
type MarketPrice struct {
	Price     float64 `json:"p"`
	Timestamp int64   `json:"t"`
}

// PriceHistoryFilterParams for price history queries
type PriceHistoryFilterParams struct {
	Market   string `json:"market,omitempty"`
	Interval string `json:"interval,omitempty"`
	StartTS  int64  `json:"startTs,omitempty"`
	EndTS    int64  `json:"endTs,omitempty"`
	Fidelity int    `json:"fidelity,omitempty"`
}

// Trade represents a trade event
type Trade struct {
	ID         string `json:"id"`
	Market     string `json:"market"`
	Asset      string `json:"asset"`
	Side       Side   `json:"side"`
	Price      string `json:"price"`
	Size       string `json:"size"`
	FeeRateBps string `json:"fee_rate_bps"`
	Timestamp  int64  `json:"timestamp"`
	Maker      string `json:"maker,omitempty"`
	Taker      string `json:"taker,omitempty"`
	MatchID    string `json:"match_id,omitempty"`
}

// TradeParams for trade queries
type TradeParams struct {
	Maker  string `json:"maker,omitempty"`
	Market string `json:"market,omitempty"`
	Asset  string `json:"asset,omitempty"`
}

// MarketTradeEvent represents live market activity
type MarketTradeEvent struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	Market    string `json:"market"`
	Asset     string `json:"asset_id"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

// Orderbook represents an L2 orderbook for a token
type Orderbook struct {
	AssetID string         `json:"asset_id"`
	Bids    []OrderSummary `json:"bids"`
	Asks    []OrderSummary `json:"asks"`
}

// OrderSummary represents a price level in the orderbook
type OrderSummary struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderArgs represents order arguments (simplified, for backward compatibility)
type OrderArgs struct {
	TokenID    string
	Side       string
	Price      string
	Size       string
	FeeRateBps string
	Nonce      string
	Expiration string
}

// OrderResponse represents response from Create/GetOrder
type OrderResponse struct {
	OrderID string      `json:"orderID"`
	Status  OrderStatus `json:"status"`
}

// ErrorResponse represents error from API
type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// PostOrdersArgs for batch order creation
type PostOrdersArgs struct {
	Order     *SignedOrderResponse `json:"order"`
	OrderType OrderType            `json:"orderType"`
	Owner     string               `json:"owner"`
}

// BatchOrderResponse for batch order creation
type BatchOrderResponse struct {
	Success []OrderResponse `json:"success,omitempty"`
	Errors  []struct {
		Index int    `json:"index"`
		Error string `json:"error"`
	} `json:"errors,omitempty"`
}

// CancelOrderResponse for single order cancellation
type CancelOrderResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// CancelOrdersResponse for batch order cancellation
type CancelOrdersResponse struct {
	Success []string `json:"success,omitempty"`
	Errors  []struct {
		OrderID string `json:"orderId"`
		Error   string `json:"error"`
	} `json:"errors,omitempty"`
}

// CancelAllResponse for cancel-all endpoint
type CancelAllResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// OrderMarketCancelParams for cancelling market orders
type OrderMarketCancelParams struct {
	Market string `json:"market,omitempty"`
	Asset  string `json:"asset,omitempty"`
}

// OpenOrderParams for querying open orders
type OpenOrderParams struct {
	Market     string `json:"market,omitempty"`
	Asset      string `json:"asset,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// OpenOrdersResponse for open orders query
type OpenOrdersResponse struct {
	NextCursor string       `json:"next_cursor"`
	Data       []*OpenOrder `json:"data"`
}

// OpenOrder represents an open order
type OpenOrder struct {
	ID              string `json:"id"`
	Market          string `json:"market"`
	Asset           string `json:"asset"`
	Owner           string `json:"owner"`
	Maker           string `json:"maker"`
	Taker           string `json:"taker"`
	Side            Side   `json:"side"`
	Price           string `json:"price"`
	OriginalSize    string `json:"original_size"`
	SizeMatched     string `json:"size_matched,omitempty"`
	Status          string `json:"status"`
	OrderType       string `json:"type,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	Expiration      string `json:"expiration,omitempty"` // Changed to string
	AssociatedOrder string `json:"associated_order,omitempty"`
}

// BalanceAllowanceParams for balance/allowance queries
type BalanceAllowanceParams struct {
	AssetType string `json:"asset_type,omitempty"`
	TokenID   string `json:"token_id,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// BalanceAllowanceResponse for balance/allowance data
type BalanceAllowanceResponse struct {
	Balance   string `json:"balance"`
	Allowance string `json:"allowance"`
}

// BanStatus represents account ban/restriction status
type BanStatus struct {
	ClosedOnly bool   `json:"closed_only"`
	Reason     string `json:"reason,omitempty"`
}

// Notification represents a user notification
type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
	Read      bool   `json:"read,omitempty"`
}

// DropNotificationParams for marking notifications as read
type DropNotificationParams struct {
	IDs []string `json:"ids,omitempty"`
}

// UserEarning represents user earnings data
type UserEarning struct {
	Day      string `json:"day"`
	Earnings string `json:"earnings"`
	Markets  int    `json:"markets,omitempty"`
}

// TotalUserEarning represents total earnings
type TotalUserEarning struct {
	Total string `json:"total"`
}

// RewardsPercentages represents reward percentage breakdown
type RewardsPercentages struct {
	Market      string    `json:"market"`
	Percentages []float64 `json:"percentages,omitempty"`
}

// MarketReward represents market reward information
type MarketReward struct {
	Market     string `json:"market"`
	RewardPool string `json:"reward_pool,omitempty"`
	Day        string `json:"day,omitempty"`
}

// UserRewardsEarning represents user rewards earning breakdown
type UserRewardsEarning struct {
	Market     string  `json:"market"`
	Day        string  `json:"day"`
	Percentage float64 `json:"percentage,omitempty"`
	Earnings   string  `json:"earnings,omitempty"`
}

// BuilderConfig for builder API configuration
type BuilderConfig struct {
	BuilderID string `json:"builder_id,omitempty"`
}

// BuilderApiKeyResponse for builder API key creation
type BuilderApiKeyResponse struct {
	APIKey string `json:"apiKey"`
	Secret string `json:"secret"`
}

// ReadonlyApiKeyResponse for readonly API key creation
type ReadonlyApiKeyResponse struct {
	APIKey string `json:"apiKey"`
}

type OrderStatus string

const (
	OrderStatusMatched   OrderStatus = "MATCHED"
	OrderStatusLive      OrderStatus = "LIVE"
	OrderStatusDelayed   OrderStatus = "DELAYED"
	OrderStatusUnmatched OrderStatus = "UNMATCHED"
)
