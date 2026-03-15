package data

// DataTrade represents a trade from the Data API (distinct from CLOB Trade)
type DataTrade struct {
	ID                    string  `json:"id,omitempty"`
	Market                string  `json:"market"` // conditionId
	Asset                 string  `json:"asset"`
	Side                  string  `json:"side"` // "BUY" | "SELL"
	Price                 float64 `json:"price"`
	Size                  float64 `json:"size"`
	Outcome               string  `json:"outcome"`
	OutcomeIndex          int     `json:"outcomeIndex"`
	Timestamp             int64   `json:"timestamp"`
	TransactionHash       string  `json:"transactionHash"`
	ProxyWallet           string  `json:"proxyWallet,omitempty"`
	Title                 string  `json:"title,omitempty"`
	Slug                  string  `json:"slug,omitempty"`
	Icon                  string  `json:"icon,omitempty"`
	EventSlug             string  `json:"eventSlug,omitempty"`
	Name                  string  `json:"name,omitempty"`
	Pseudonym             string  `json:"pseudonym,omitempty"`
	Bio                   string  `json:"bio,omitempty"`
	ProfileImage          string  `json:"profileImage,omitempty"`
	ProfileImageOptimized string  `json:"profileImageOptimized,omitempty"`
}

// Position represents a position from Data API
type Position struct {
	ProxyWallet        string  `json:"proxyWallet,omitempty"`
	Asset              string  `json:"asset"`
	ConditionID        string  `json:"conditionId"`
	Outcome            string  `json:"outcome"`
	OutcomeIndex       int     `json:"outcomeIndex"`
	Size               float64 `json:"size"`
	AvgPrice           float64 `json:"avgPrice"`
	CurPrice           float64 `json:"curPrice,omitempty"`
	TotalBought        float64 `json:"totalBought,omitempty"`
	InitialValue       float64 `json:"initialValue,omitempty"`
	CurrentValue       float64 `json:"currentValue,omitempty"`
	CashPnl            float64 `json:"cashPnl,omitempty"`
	PercentPnl         float64 `json:"percentPnl,omitempty"`
	RealizedPnl        float64 `json:"realizedPnl,omitempty"`
	PercentRealizedPnl float64 `json:"percentRealizedPnl,omitempty"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug,omitempty"`
	Icon               string  `json:"icon,omitempty"`
	EventID            string  `json:"eventId,omitempty"`
	EventSlug          string  `json:"eventSlug,omitempty"`
	OppositeOutcome    string  `json:"oppositeOutcome,omitempty"`
	OppositeAsset      string  `json:"oppositeAsset,omitempty"`
	Redeemable         bool    `json:"redeemable,omitempty"`
	Mergeable          bool    `json:"mergeable,omitempty"`
	EndDate            string  `json:"endDate,omitempty"`
	NegativeRisk       bool    `json:"negativeRisk,omitempty"`
}

// Activity represents an activity entry from Data API
type Activity struct {
	Type            string  `json:"type"` // "TRADE" | "SPLIT" | "MERGE" | "REDEEM" | "REWARD" | "CONVERSION"
	Side            string  `json:"side"` // "BUY" | "SELL"
	Size            float64 `json:"size"`
	Price           float64 `json:"price"`
	UsdcSize        float64 `json:"usdcSize,omitempty"`
	Asset           string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	Outcome         string  `json:"outcome"`
	OutcomeIndex    int     `json:"outcomeIndex,omitempty"`
	Timestamp       int64   `json:"timestamp"`
	TransactionHash string  `json:"transactionHash"`
	ProxyWallet     string  `json:"proxyWallet,omitempty"`
	Title           string  `json:"title,omitempty"`
	Slug            string  `json:"slug,omitempty"`
	Name            string  `json:"name,omitempty"`
}

// ClosedPosition represents a closed position entry
type ClosedPosition struct {
	ProxyWallet     string  `json:"proxyWallet"`
	Asset           string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	AvgPrice        float64 `json:"avgPrice"`
	TotalBought     float64 `json:"totalBought"`
	RealizedPnl     float64 `json:"realizedPnl"`
	CurPrice        float64 `json:"curPrice"`
	Timestamp       int64   `json:"timestamp"`
	Title           string  `json:"title"`
	Slug            string  `json:"slug,omitempty"`
	Icon            string  `json:"icon,omitempty"`
	EventSlug       string  `json:"eventSlug,omitempty"`
	Outcome         string  `json:"outcome"`
	OutcomeIndex    int     `json:"outcomeIndex"`
	OppositeOutcome string  `json:"oppositeOutcome,omitempty"`
	OppositeAsset   string  `json:"oppositeAsset,omitempty"`
	EndDate         string  `json:"endDate,omitempty"`
}

// LeaderboardEntry represents a leaderboard entry
type LeaderboardEntry struct {
	Address       string  `json:"address"` // Standardized from proxyWallet
	Rank          int     `json:"rank"`    // Converted from string in API if needed, but assuming int in struct
	Pnl           float64 `json:"pnl"`
	Volume        float64 `json:"volume"` // Renamed from vol
	UserName      string  `json:"userName,omitempty"`
	XUsername     string  `json:"xUsername,omitempty"`
	VerifiedBadge bool    `json:"verifiedBadge,omitempty"`
	ProfileImage  string  `json:"profileImage,omitempty"`
	Positions     int     `json:"positions,omitempty"`
	Trades        int     `json:"trades,omitempty"`
}

// APILeaderboardEntry represents raw API response for leaderboard
type APILeaderboardEntry struct {
	Rank          interface{} `json:"rank"` // Can be string or number
	ProxyWallet   string      `json:"proxyWallet"`
	UserName      string      `json:"userName"`
	Vol           float64     `json:"vol"`
	Pnl           float64     `json:"pnl"`
	ProfileImage  string      `json:"profileImage"`
	XUsername     string      `json:"xUsername"`
	VerifiedBadge bool        `json:"verifiedBadge"`
}

// LeaderboardResult represents the result with metadata
type LeaderboardResult struct {
	Entries []LeaderboardEntry `json:"entries"`
	HasMore bool               `json:"hasMore"`
	Request LeaderboardParams  `json:"request"`
}

// Query Parameters

type PositionsParams struct {
	Limit         int      `url:"limit,omitempty"`
	Offset        int      `url:"offset,omitempty"`
	SortBy        string   `url:"sortBy,omitempty"`        // CURRENT, INITIAL, TOKENS, CASHPNL, PERCENTPNL, TITLE, RESOLVING, PRICE, AVGPRICE
	SortDirection string   `url:"sortDirection,omitempty"` // ASC, DESC
	Market        []string `url:"market,omitempty"`
	EventID       []int    `url:"eventId,omitempty"`
	SizeThreshold float64  `url:"sizeThreshold,omitempty"`
	Redeemable    *bool    `url:"redeemable,omitempty"`
	Mergeable     *bool    `url:"mergeable,omitempty"`
	Title         string   `url:"title,omitempty"`
}

type ClosedPositionsParams struct {
	Limit         int      `url:"limit,omitempty"`
	Offset        int      `url:"offset,omitempty"`
	Market        []string `url:"market,omitempty"`
	EventID       []int    `url:"eventId,omitempty"`
	Title         string   `url:"title,omitempty"`
	SortBy        string   `url:"sortBy,omitempty"`        // REALIZEDPNL, TITLE, PRICE, AVGPRICE, TIMESTAMP
	SortDirection string   `url:"sortDirection,omitempty"` // ASC, DESC
}

type ActivityParams struct {
	Limit         int      `url:"limit,omitempty"`
	Offset        int      `url:"offset,omitempty"`
	Start         int64    `url:"start,omitempty"`
	End           int64    `url:"end,omitempty"`
	Type          string   `url:"type,omitempty"` // TRADE, SPLIT, MERGE, REDEEM, REWARD, CONVERSION
	Side          string   `url:"side,omitempty"` // BUY, SELL
	Market        []string `url:"market,omitempty"`
	EventID       []int    `url:"eventId,omitempty"`
	SortBy        string   `url:"sortBy,omitempty"`        // TIMESTAMP, TOKENS, CASH
	SortDirection string   `url:"sortDirection,omitempty"` // ASC, DESC
}

type TradesParams struct {
	Limit          int     `url:"limit,omitempty"`
	Market         string  `url:"market,omitempty"`
	User           string  `url:"user,omitempty"`
	TakerOnly      *bool   `url:"takerOnly,omitempty"`
	FilterType     string  `url:"filterType,omitempty"` // CASH | TOKENS
	FilterAmount   float64 `url:"filterAmount,omitempty"`
	Side           string  `url:"side,omitempty"` // BUY | SELL
	StartTimestamp int64   `url:"-"`              // Client-side filtering
	EndTimestamp   int64   `url:"-"`              // Client-side filtering
}

type LeaderboardParams struct {
	TimePeriod string `url:"timePeriod,omitempty"` // DAY, WEEK, MONTH, ALL
	OrderBy    string `url:"orderBy,omitempty"`    // PNL, VOL
	Category   string `url:"category,omitempty"`   // OVERALL, POLITICS, SPORTS, CRYPTO, CULTURE, MENTIONS, WEATHER, ECONOMICS, TECH, FINANCE
	Limit      int    `url:"limit,omitempty"`
	Offset     int    `url:"offset,omitempty"`
	User       string `url:"user,omitempty"`
	UserName   string `url:"userName,omitempty"`
}
