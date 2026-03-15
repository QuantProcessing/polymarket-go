package clob

import (
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	orderutils "github.com/polymarket/go-order-utils/pkg/model"
)

const (
	CollateralTokenDecimals = 6
	ZeroAddress             = "0x0000000000000000000000000000000000000000"
)

type TickSize string

const (
	TickSize01    TickSize = "0.1"
	TickSize001   TickSize = "0.01"
	TickSize0001  TickSize = "0.001"
	TickSize00001 TickSize = "0.0001"
)

type RoundConfig struct {
	Price  int
	Size   int
	Amount int
}

var RoundingConfigs = map[TickSize]RoundConfig{
	TickSize01:    {Price: 1, Size: 2, Amount: 3},
	TickSize001:   {Price: 2, Size: 2, Amount: 4},
	TickSize0001:  {Price: 3, Size: 2, Amount: 5},
	TickSize00001: {Price: 4, Size: 2, Amount: 6},
}

type OrderBuilder struct {
	signer  *Signer
	chainID int64
}

func NewOrderBuilder(signer *Signer, chainID int64) *OrderBuilder {
	return &OrderBuilder{
		signer:  signer,
		chainID: chainID,
	}
}

type UserOrderParams struct {
	TokenID    string
	Price      float64
	Size       float64
	Side       Side
	FeeRateBps int
	Nonce      int64
	Expiration int64
	Taker      string
}

type UserMarketOrderParams struct {
	TokenID    string
	Amount     float64 // USDC for BUY, Tokens for SELL
	Side       Side
	Price      *float64 // Optional limit price protection
	FeeRateBps int
	Nonce      int64
	Taker      string
	OrderType  OrderType // e.g. FOK, FAK, IOC
}

type BuildOrderOptions struct {
	TickSize string
	NegRisk  bool
	Maker    string
}

// BuildOrder creates and signs a limit order using official go-order-utils
func (b *OrderBuilder) BuildOrder(params UserOrderParams, opts BuildOrderOptions) (*SignedOrderResponse, error) {
	tickSize := TickSize(NormalizeTickSize(opts.TickSize))
	roundConfig, ok := RoundingConfigs[tickSize]
	if !ok {
		return nil, fmt.Errorf("invalid tick size: %s (normalized: %s)", opts.TickSize, tickSize)
	}

	if !IsValidPrice(params.Price, tickSize) {
		return nil, fmt.Errorf("invalid price %.4f for tick size %s", params.Price, opts.TickSize)
	}

	maker := opts.Maker
	if maker == "" {
		maker = b.signer.GetAddress()
	}

	// Calculate amounts
	rawMakerAmt, rawTakerAmt := calculateOrderAmounts(params.Side, params.Size, params.Price, roundConfig)
	return b.buildOrderArgs(params.TokenID, maker, params.Taker, params.Side, params.FeeRateBps, params.Nonce, params.Expiration, rawMakerAmt, rawTakerAmt, opts)
}

// BuildMarketOrder builds a market order (amount based)
func (b *OrderBuilder) BuildMarketOrder(params UserMarketOrderParams, opts BuildOrderOptions) (*SignedOrderResponse, error) {
	tickSize := TickSize(NormalizeTickSize(opts.TickSize))
	roundConfig, ok := RoundingConfigs[tickSize]
	if !ok {
		return nil, fmt.Errorf("invalid tick size: %s", opts.TickSize)
	}

	price := 1.0
	if params.Price != nil {
		price = *params.Price
	}

	if !IsValidPrice(price, tickSize) {
		return nil, fmt.Errorf("invalid price %.4f for tick size %s", price, opts.TickSize)
	}

	maker := opts.Maker
	if maker == "" {
		maker = b.signer.GetAddress()
	}

	rawMakerAmt, rawTakerAmt := calculateMarketOrderAmounts(params.Side, params.Amount, price, roundConfig)

	// Default FOK/IOC for market orders? Only affects signature if API requires it?
	// Actually the Order Data structure is same. Order Type is sent in payload.
	// Expiration 0 for GTC/FOK usually.

	return b.buildOrderArgs(params.TokenID, maker, params.Taker, params.Side, params.FeeRateBps, params.Nonce, 0, rawMakerAmt, rawTakerAmt, opts)
}

func (b *OrderBuilder) buildOrderArgs(tokenID, maker, taker string, sideParam Side, feeRateBps int, nonce, expiration int64, rawMakerAmt, rawTakerAmt float64, opts BuildOrderOptions) (*SignedOrderResponse, error) {
	makerAmountWei := toWei(rawMakerAmt, CollateralTokenDecimals)
	takerAmountWei := toWei(rawTakerAmt, CollateralTokenDecimals)

	if taker == "" {
		taker = ZeroAddress
	}

	// Convert Side to model.Side enum
	var side orderutils.Side
	if sideParam == SideBuy {
		side = orderutils.BUY
	} else {
		side = orderutils.SELL
	}

	// Determine SignatureType
	signatureType := orderutils.EOA
	if !strings.EqualFold(maker, b.signer.GetAddress()) {
		signatureType = orderutils.POLY_GNOSIS_SAFE
	}

	// Build OrderData for official builder
	orderData := &orderutils.OrderData{
		Maker:         maker,
		Signer:        b.signer.GetAddress(),
		Taker:         taker,
		TokenId:       tokenID,
		MakerAmount:   makerAmountWei,
		TakerAmount:   takerAmountWei,
		Side:          side,
		FeeRateBps:    fmt.Sprintf("%d", feeRateBps),
		Nonce:         fmt.Sprintf("%d", nonce),
		Expiration:    fmt.Sprintf("%d", expiration),
		SignatureType: signatureType,
	}

	// Select contract type
	contract := orderutils.CTFExchange
	if opts.NegRisk {
		contract = orderutils.NegRiskCTFExchange
	}

	// Build order using official go-order-utils
	signedOrder, err := b.signer.SignOrder(orderData, contract)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	// Convert side back to string for JSON
	sideStr := "BUY"
	if signedOrder.Side.Int64() == 1 {
		sideStr = "SELL"
	}

	// Convert to JSON format matching TypeScript SDK
	return &SignedOrderResponse{
		Salt:          signedOrder.Salt.Int64(),
		Maker:         signedOrder.Maker.Hex(),
		Signer:        signedOrder.Signer.Hex(),
		Taker:         signedOrder.Taker.Hex(),
		TokenId:       signedOrder.TokenId.String(),
		MakerAmount:   signedOrder.MakerAmount.String(),
		TakerAmount:   signedOrder.TakerAmount.String(),
		Side:          sideStr,
		Expiration:    signedOrder.Expiration.String(),
		Nonce:         signedOrder.Nonce.String(),
		FeeRateBps:    signedOrder.FeeRateBps.String(),
		SignatureType: int(signedOrder.SignatureType.Int64()),
		Signature:     "0x" + common.Bytes2Hex(signedOrder.Signature),
	}, nil
}

// SignedOrderResponse matches TypeScript SDK's NewOrder.order structure
type SignedOrderResponse struct {
	Salt          int64  `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	Taker         string `json:"taker"`
	TokenId       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          string `json:"side"`
	Expiration    string `json:"expiration"`
	Nonce         string `json:"nonce"`
	FeeRateBps    string `json:"feeRateBps"`
	SignatureType int    `json:"signatureType"`
	Signature     string `json:"signature"`
}

// Helper functions

func calculateOrderAmounts(side Side, size, price float64, config RoundConfig) (float64, float64) {
	rawPrice := roundNormal(price, config.Price)
	if side == SideBuy {
		// BUY: Maker gives USDC, Taker gives Tokens
		// Price = TakerAmount / MakerAmount = Tokens / USDC
		// For price 0.3: you pay 0.3 USDC per token
		rawTakerAmt := roundDown(size, config.Size) // tokens you receive
		rawMakerAmt := rawTakerAmt * rawPrice       // USDC you pay
		if decimalPlaces(rawMakerAmt) > config.Amount {
			rawMakerAmt = roundUp(rawMakerAmt, config.Amount+4)
			if decimalPlaces(rawMakerAmt) > config.Amount {
				rawMakerAmt = roundDown(rawMakerAmt, config.Amount)
			}
		}
		return rawMakerAmt, rawTakerAmt
	}

	// SELL: Maker gives Tokens, Taker gives USDC
	// Price = TakerAmount / MakerAmount = USDC / Tokens
	// For price 0.4: you receive 0.4 USDC per token
	rawMakerAmt := roundDown(size, config.Size) // tokens you sell (maker)
	rawTakerAmt := rawMakerAmt * rawPrice       // USDC you receive (taker)
	if decimalPlaces(rawTakerAmt) > config.Amount {
		rawTakerAmt = roundUp(rawTakerAmt, config.Amount+4)
		if decimalPlaces(rawTakerAmt) > config.Amount {
			rawTakerAmt = roundDown(rawTakerAmt, config.Amount)
		}
	}
	return rawMakerAmt, rawTakerAmt
}

func calculateMarketOrderAmounts(side Side, amount, price float64, config RoundConfig) (float64, float64) {
	// Reference: clob-client/src/order-builder/helpers.ts getMarketOrderRawAmounts
	rawPrice := roundNormal(price, config.Price)

	if side == SideBuy {
		// MARKET BUY: Amount is USDC (Maker)
		// We want to spend `amount` USDC
		// rawMakerAmt (USDC) = roundDown(amount)
		// rawTakerAmt (Tokens) = rawMakerAmt / price

		rawMakerAmt := roundDown(amount, config.Size) // Round USDC amount to 2 decimals (config.Size for tick 0.01 is 2)
		rawTakerAmt := rawMakerAmt / rawPrice         // Tokens receiving

		if decimalPlaces(rawTakerAmt) > config.Amount {
			rawTakerAmt = roundUp(rawTakerAmt, config.Amount+4)
			if decimalPlaces(rawTakerAmt) > config.Amount {
				rawTakerAmt = roundDown(rawTakerAmt, config.Amount)
			}
		}
		return rawMakerAmt, rawTakerAmt
	}

	// MARKET SELL: Amount is Tokens (Maker)
	// We want to sell `amount` Tokens
	// rawMakerAmt (Tokens) = roundDown(amount)
	// rawTakerAmt (USDC) = rawMakerAmt * price

	rawMakerAmt := roundDown(amount, config.Size) // Round Token amount
	rawTakerAmt := rawMakerAmt * rawPrice         // USDC receiving

	if decimalPlaces(rawTakerAmt) > config.Amount {
		rawTakerAmt = roundUp(rawTakerAmt, config.Amount+4)
		if decimalPlaces(rawTakerAmt) > config.Amount {
			rawTakerAmt = roundDown(rawTakerAmt, config.Amount)
		}
	}
	return rawMakerAmt, rawTakerAmt
}

func roundNormal(num float64, decimals int) float64 {
	if decimalPlaces(num) <= decimals {
		return num
	}
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(num*multiplier) / multiplier
}

func roundDown(num float64, decimals int) float64 {
	if decimalPlaces(num) <= decimals {
		return num
	}
	multiplier := math.Pow(10, float64(decimals))
	return math.Floor(num*multiplier) / multiplier
}

func roundUp(num float64, decimals int) float64 {
	if decimalPlaces(num) <= decimals {
		return num
	}
	multiplier := math.Pow(10, float64(decimals))
	return math.Ceil(num*multiplier) / multiplier
}

func decimalPlaces(num float64) int {
	s := fmt.Sprintf("%.10f", num)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return 0
	}
	return len(parts[1])
}

func toWei(amount float64, decimals int) string {
	multiplier := math.Pow(10, float64(decimals))
	result := amount * multiplier
	return fmt.Sprintf("%.0f", result)
}

func IsValidPrice(price float64, tickSize TickSize) bool {
	ts := parseTickSize(string(tickSize))
	return price >= ts && price <= 1-ts
}

func parseTickSize(s string) float64 {
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// NormalizeTickSize normalizes API tick size format to builder format
// "0.0100" -> "0.01", "0.1000" -> "0.1"
func NormalizeTickSize(tickSize string) string {
	val := parseTickSize(tickSize)
	if val == 0 {
		return "0.01" // default
	}
	// Format to remove trailing zeros
	normalized := fmt.Sprintf("%.4f", val)
	normalized = strings.TrimRight(normalized, "0")
	normalized = strings.TrimRight(normalized, ".")
	return normalized
}
