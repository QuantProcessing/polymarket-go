package clob

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// IsTickSizeSmaller compares two tick sizes
func IsTickSizeSmaller(a, b string) bool {
	aFloat, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return false
	}

	bFloat, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return false
	}

	return aFloat < bFloat
}

// GenerateOrderBookSummaryHash generates a hash for orderbook summary
func GenerateOrderBookSummaryHash(summary *OrderBookSummary) string {
	// Convert to JSON for consistent hashing
	data, err := json.Marshal(summary)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// OrderToJSON converts an order to JSON string
func OrderToJSON(order *SignedOrderResponse) (string, error) {
	data, err := json.Marshal(order)
	if err != nil {
		return "", fmt.Errorf("failed to marshal order: %w", err)
	}
	return string(data), nil
}

// ParseFloatSafe safely parses a float from string
func ParseFloatSafe(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// FormatPrice formats price with appropriate precision
func FormatPrice(price float64, precision int) string {
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, price)
}

// RoundToTickSize rounds a price to the nearest tick size
func RoundToTickSize(price float64, tickSize string) float64 {
	tickFloat, err := strconv.ParseFloat(tickSize, 64)
	if err != nil {
		return price
	}

	return math.Round(price/tickFloat) * tickFloat
}
