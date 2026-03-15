package subgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Subgraph Endpoints
const (
	SubgraphPositions = "https://api.goldsky.com/api/public/project_cl6mb8i9h0003e201j6li0diw/subgraphs/positions-subgraph/0.0.7/gn"
	SubgraphPnL       = "https://api.goldsky.com/api/public/project_cl6mb8i9h0003e201j6li0diw/subgraphs/pnl-subgraph/0.0.14/gn"
	SubgraphActivity  = "https://api.goldsky.com/api/public/project_cl6mb8i9h0003e201j6li0diw/subgraphs/activity-subgraph/0.0.4/gn"
	SubgraphOI        = "https://api.goldsky.com/api/public/project_cl6mb8i9h0003e201j6li0diw/subgraphs/oi-subgraph/0.0.6/gn"
	SubgraphOrderbook = "https://api.goldsky.com/api/public/project_cl6mb8i9h0003e201j6li0diw/subgraphs/orderbook-subgraph/0.0.1/gn"
)

type SubgraphName string

const (
	SubgraphPos           SubgraphName = "positions"
	SubgraphPnLType       SubgraphName = "pnl"
	SubgraphAct           SubgraphName = "activity"
	SubgraphOIType        SubgraphName = "oi"
	SubgraphOrderbookType SubgraphName = "orderbook"
)

var SubgraphEndpoints = map[SubgraphName]string{
	SubgraphPos:           SubgraphPositions,
	SubgraphPnLType:       SubgraphPnL,
	SubgraphAct:           SubgraphActivity,
	SubgraphOIType:        SubgraphOI,
	SubgraphOrderbookType: SubgraphOrderbook,
}

// --- Types ---

// Positions Subgraph
type UserBalance struct {
	ID      string `json:"id"`
	User    string `json:"user"`
	Asset   string `json:"asset"`
	Balance string `json:"balance"` // BigInt as string
}

type NetUserBalance struct {
	ID      string `json:"id"`
	User    string `json:"user"`
	Asset   string `json:"asset"`
	Balance string `json:"balance"`
}

// PnL Subgraph
type UserPosition struct {
	ID          string `json:"id"`
	User        string `json:"user"`
	TokenID     string `json:"tokenId"` // BigInt as string
	Amount      string `json:"amount"`
	AvgPrice    string `json:"avgPrice"`
	RealizedPnl string `json:"realizedPnl"`
	TotalBought string `json:"totalBought"`
}

type Condition struct {
	ID                string   `json:"id"`
	PositionIDs       []string `json:"positionIds"`
	PayoutNumerators  []string `json:"payoutNumerators"`
	PayoutDenominator string   `json:"payoutDenominator"`
}

// Activity Subgraph
type Split struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	Stakeholder string `json:"stakeholder"`
	Condition   string `json:"condition"`
	Amount      string `json:"amount"`
}

type Merge struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	Stakeholder string `json:"stakeholder"`
	Condition   string `json:"condition"`
	Amount      string `json:"amount"`
}

type Redemption struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Redeemer  string `json:"redeemer"`
	Condition string `json:"condition"`
	Payout    string `json:"payout"`
}

// OI Subgraph
type MarketOpenInterest struct {
	ID     string `json:"id"` // condition id
	Amount string `json:"amount"`
}

type GlobalOpenInterest struct {
	ID     string `json:"id"`
	Amount string `json:"amount"`
}

// Orderbook Subgraph
type OrderFilledEvent struct {
	ID                string `json:"id"`
	TransactionHash   string `json:"transactionHash"`
	Timestamp         string `json:"timestamp"`
	OrderHash         string `json:"orderHash"`
	Maker             string `json:"maker"`
	Taker             string `json:"taker"`
	MakerAssetID      string `json:"makerAssetId"`
	TakerAssetID      string `json:"takerAssetId"`
	MakerAmountFilled string `json:"makerAmountFilled"`
	TakerAmountFilled string `json:"takerAmountFilled"`
	Fee               string `json:"fee"`
}

type MarketData struct {
	ID     string `json:"id"`
	Volume string `json:"volume"`
}

// Query Parameters
type SubgraphQueryParams struct {
	First          *int                   `json:"first,omitempty"`
	Skip           *int                   `json:"skip,omitempty"`
	OrderBy        string                 `json:"orderBy,omitempty"`
	OrderDirection string                 `json:"orderDirection,omitempty"` // "asc" | "desc"
	Where          map[string]interface{} `json:"where,omitempty"`
}

// --- Client ---

type SubgraphClient struct {
	HTTPClient *http.Client
}

func NewSubgraphClient(httpClient *http.Client) *SubgraphClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &SubgraphClient{HTTPClient: httpClient}
}

// Generic Query wrapper
type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *SubgraphClient) query(subgraph SubgraphName, queryStr string, result interface{}) error {
	endpoint := SubgraphEndpoints[subgraph]
	reqBody, err := json.Marshal(graphQLRequest{Query: queryStr})
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := c.HTTPClient.Post(endpoint, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("subgraph request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// fmt.Printf("DEBUG Subgraph Query: %s\nResponse: %s\n", queryStr, string(body)) // Debugging

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subgraph request failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var qr graphQLResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(qr.Errors) > 0 {
		var msgs []string
		for _, e := range qr.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("graphql error: %s", strings.Join(msgs, ", "))
	}

	if result != nil {
		if err := json.Unmarshal(qr.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}
	return nil
}

func (c *SubgraphClient) buildQuery(entityName string, fields []string, params SubgraphQueryParams) string {
	var args []string

	if params.First != nil {
		args = append(args, fmt.Sprintf("first: %d", *params.First))
	}
	if params.Skip != nil {
		args = append(args, fmt.Sprintf("skip: %d", *params.Skip))
	}
	if params.OrderBy != "" {
		args = append(args, fmt.Sprintf("orderBy: %s", params.OrderBy))
	}
	if params.OrderDirection != "" {
		args = append(args, fmt.Sprintf("orderDirection: %s", params.OrderDirection))
	}
	if len(params.Where) > 0 {
		var whereParts []string
		for k, v := range params.Where {
			switch val := v.(type) {
			case string:
				whereParts = append(whereParts, fmt.Sprintf("%s: \"%s\"", k, val))
			case int, int64, float64, bool:
				whereParts = append(whereParts, fmt.Sprintf("%s: %v", k, val))
			default:
				// Fallback to json stringify for complex types (arrays, etc)
				b, _ := json.Marshal(val)
				whereParts = append(whereParts, fmt.Sprintf("%s: %s", k, string(b)))
			}
		}
		args = append(args, fmt.Sprintf("where: { %s }", strings.Join(whereParts, ", ")))
	}

	argsStr := ""
	if len(args) > 0 {
		argsStr = fmt.Sprintf("(%s)", strings.Join(args, ", "))
	}

	fieldsStr := strings.Join(fields, "\n        ")

	return fmt.Sprintf(`{
    %s%s {
        %s
    }
}`, entityName, argsStr, fieldsStr)
}

// --- Methods ---

// User Balances
func (c *SubgraphClient) GetUserBalances(user string, params SubgraphQueryParams) ([]UserBalance, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["user"] = strings.ToLower(user)

	query := c.buildQuery("userBalances", []string{"id", "user", "asset", "balance"}, params)
	var res struct {
		UserBalances []UserBalance `json:"userBalances"`
	}
	if err := c.query(SubgraphPos, query, &res); err != nil {
		return nil, err
	}
	return res.UserBalances, nil
}

func (c *SubgraphClient) GetNetUserBalances(user string, params SubgraphQueryParams) ([]NetUserBalance, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["user"] = strings.ToLower(user)

	query := c.buildQuery("netUserBalances", []string{"id", "user", "asset", "balance"}, params)
	var res struct {
		NetUserBalances []NetUserBalance `json:"netUserBalances"`
	}
	if err := c.query(SubgraphPos, query, &res); err != nil {
		return nil, err
	}
	return res.NetUserBalances, nil
}

// PnL
func (c *SubgraphClient) GetUserPositions(user string, params SubgraphQueryParams) ([]UserPosition, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["user"] = strings.ToLower(user)
	if params.OrderBy == "" {
		params.OrderBy = "realizedPnl"
		params.OrderDirection = "desc"
	}

	query := c.buildQuery("userPositions", []string{"id", "user", "tokenId", "amount", "avgPrice", "realizedPnl", "totalBought"}, params)
	var res struct {
		UserPositions []UserPosition `json:"userPositions"`
	}
	if err := c.query(SubgraphPnLType, query, &res); err != nil {
		return nil, err
	}
	return res.UserPositions, nil
}

func (c *SubgraphClient) GetConditions(params SubgraphQueryParams) ([]Condition, error) {
	query := c.buildQuery("conditions", []string{"id", "positionIds", "payoutNumerators", "payoutDenominator"}, params)
	var res struct {
		Conditions []Condition `json:"conditions"`
	}
	if err := c.query(SubgraphPnLType, query, &res); err != nil {
		return nil, err
	}
	return res.Conditions, nil
}

func (c *SubgraphClient) GetCondition(conditionID string) (*Condition, error) {
	query := fmt.Sprintf(`{
		condition(id: "%s") {
			id
			positionIds
			payoutNumerators
			payoutDenominator
		}
	}`, strings.ToLower(conditionID))

	var res struct {
		Condition *Condition `json:"condition"`
	}
	if err := c.query(SubgraphPnLType, query, &res); err != nil {
		return nil, err
	}
	return res.Condition, nil
}

func (c *SubgraphClient) IsConditionResolved(conditionID string) (bool, error) {
	cond, err := c.GetCondition(conditionID)
	if err != nil {
		return false, err
	}
	if cond == nil {
		return false, nil
	}
	return len(cond.PayoutNumerators) > 0 && cond.PayoutDenominator != "0", nil
}

// Activity
func (c *SubgraphClient) GetSplits(user string, params SubgraphQueryParams) ([]Split, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["stakeholder"] = strings.ToLower(user)
	if params.OrderBy == "" {
		params.OrderBy = "timestamp"
		params.OrderDirection = "desc"
	}

	query := c.buildQuery("splits", []string{"id", "timestamp", "stakeholder", "condition", "amount"}, params)
	var res struct {
		Splits []Split `json:"splits"`
	}
	if err := c.query(SubgraphAct, query, &res); err != nil {
		return nil, err
	}
	return res.Splits, nil
}

func (c *SubgraphClient) GetMerges(user string, params SubgraphQueryParams) ([]Merge, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["stakeholder"] = strings.ToLower(user)
	if params.OrderBy == "" {
		params.OrderBy = "timestamp"
		params.OrderDirection = "desc"
	}

	query := c.buildQuery("merges", []string{"id", "timestamp", "stakeholder", "condition", "amount"}, params)
	var res struct {
		Merges []Merge `json:"merges"`
	}
	if err := c.query(SubgraphAct, query, &res); err != nil {
		return nil, err
	}
	return res.Merges, nil
}

func (c *SubgraphClient) GetRedemptions(user string, params SubgraphQueryParams) ([]Redemption, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["redeemer"] = strings.ToLower(user)
	if params.OrderBy == "" {
		params.OrderBy = "timestamp"
		params.OrderDirection = "desc"
	}

	query := c.buildQuery("redemptions", []string{"id", "timestamp", "redeemer", "condition", "payout"}, params)
	var res struct {
		Redemptions []Redemption `json:"redemptions"`
	}
	if err := c.query(SubgraphAct, query, &res); err != nil {
		return nil, err
	}
	return res.Redemptions, nil
}

// OI
func (c *SubgraphClient) GetMarketOpenInterest(conditionID string) (*MarketOpenInterest, error) {
	query := fmt.Sprintf(`{
		marketOpenInterest(id: "%s") {
			id
			amount
		}
	}`, strings.ToLower(conditionID))

	var res struct {
		MarketOpenInterest *MarketOpenInterest `json:"marketOpenInterest"`
	}
	if err := c.query(SubgraphOIType, query, &res); err != nil {
		return nil, err
	}
	return res.MarketOpenInterest, nil
}

func (c *SubgraphClient) GetTopMarketsByOI(params SubgraphQueryParams) ([]MarketOpenInterest, error) {
	if params.OrderBy == "" {
		params.OrderBy = "amount"
		params.OrderDirection = "desc"
	}
	if params.First == nil {
		defaultLimit := 50
		params.First = &defaultLimit
	}

	query := c.buildQuery("marketOpenInterests", []string{"id", "amount"}, params)
	var res struct {
		MarketOpenInterests []MarketOpenInterest `json:"marketOpenInterests"`
	}
	if err := c.query(SubgraphOIType, query, &res); err != nil {
		return nil, err
	}
	return res.MarketOpenInterests, nil
}

// Orderbook
func (c *SubgraphClient) GetOrderFilledEvents(params SubgraphQueryParams) ([]OrderFilledEvent, error) {
	if params.OrderBy == "" {
		params.OrderBy = "timestamp"
		params.OrderDirection = "desc"
	}
	if params.First == nil {
		defaultLimit := 100
		params.First = &defaultLimit
	}

	query := c.buildQuery("orderFilledEvents", []string{"id", "transactionHash", "timestamp", "orderHash", "maker", "taker", "makerAssetId", "takerAssetId", "makerAmountFilled", "takerAmountFilled", "fee"}, params)
	var res struct {
		OrderFilledEvents []OrderFilledEvent `json:"orderFilledEvents"`
	}
	if err := c.query(SubgraphOrderbookType, query, &res); err != nil {
		return nil, err
	}
	return res.OrderFilledEvents, nil
}

func (c *SubgraphClient) GetMakerFills(maker string, params SubgraphQueryParams) ([]OrderFilledEvent, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["maker"] = strings.ToLower(maker)
	return c.GetOrderFilledEvents(params)
}

func (c *SubgraphClient) GetTakerFills(taker string, params SubgraphQueryParams) ([]OrderFilledEvent, error) {
	if params.Where == nil {
		params.Where = make(map[string]interface{})
	}
	params.Where["taker"] = strings.ToLower(taker)
	return c.GetOrderFilledEvents(params)
}

// Helpers
func IntP(i int) *int {
	return &i
}
