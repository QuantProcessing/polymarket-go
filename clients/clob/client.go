package clob

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/QuantProcessing/polymarket-go/prediction"

	"github.com/ethereum/go-ethereum/crypto"

	"go.uber.org/zap"
)

const (
	BaseURL = "https://clob.polymarket.com"
)

type ClobClient struct {
	baseURL      string
	httpClient   *http.Client
	logger       *zap.SugaredLogger
	creds        *Credentials
	signer       *Signer
	orderBuilder *OrderBuilder // Order builder for creating and signing orders
}

type Credentials struct {
	APIKey        string
	APISecret     string
	APIPassphrase string
	PrivateKey    string
	FunderAddress string
	ChainID       int64
}

func NewClient(baseURL string, logger *zap.SugaredLogger) *ClobClient {
	if baseURL == "" {
		baseURL = BaseURL
	}
	return &ClobClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *ClobClient) WithCredentials(creds *Credentials) *ClobClient {
	c.creds = creds

	chainID := creds.ChainID
	if chainID == 0 {
		chainID = 137 // Default to Polygon Mainnet
	}

	// Initialize signer if we have a private key
	if creds.PrivateKey != "" {
		signer, err := NewSigner(creds.PrivateKey, chainID)
		if err != nil {
			c.logger.Errorw("Failed to create signer", "error", err)
		} else {
			c.signer = signer
			// Initialize order builder
			c.orderBuilder = NewOrderBuilder(signer, chainID)
		}
	}
	return c
}

// EnsureAuth makes sure we have L2 credentials (API Credentials)
// If they are missing but we have a Private Key, we try to derive them.
func (c *ClobClient) EnsureAuth(ctx context.Context) error {
	if c.creds == nil {
		return fmt.Errorf("no credentials provided")
	}
	if c.creds.APIKey != "" {
		return nil // Already authenticated
	}
	if c.signer == nil {
		return fmt.Errorf("cannot derive API key: private key missing")
	}

	return c.deriveAPIKey(ctx)
}

type DeriveKeyResponse struct {
	ApiKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

func (c *ClobClient) deriveAPIKey(ctx context.Context) error {
	c.logger.Info("Deriving API Credentials from Private Key (L1 -> L2)...")

	// Fetch Server Time
	// Using a direct call to the time endpoint, not a helper, as this is part of auth setup.
	urlTime := fmt.Sprintf("%s/time", c.baseURL)
	reqTime, err := http.NewRequestWithContext(ctx, "GET", urlTime, nil)
	if err != nil {
		return fmt.Errorf("failed to create time request: %w", err)
	}
	var serverTimeRes interface{}
	if err := c.do(reqTime, &serverTimeRes, false); err != nil {
		c.logger.Warnw("Failed to get server time, using local", "error", err)
	}

	var ts int64
	switch v := serverTimeRes.(type) {
	case float64:
		ts = int64(v)
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err == nil {
			ts = parsed.Unix()
		}
	case map[string]interface{}:
		if t, ok := v["server_time"].(string); ok {
			parsed, err := time.Parse(time.RFC3339, t)
			if err == nil {
				ts = parsed.Unix()
			} else {
				parsed, err = time.Parse("2006-01-02T15:04:05.000Z", t)
				if err == nil {
					ts = parsed.Unix()
				}
			}
		} else if t, ok := v["server_time"].(float64); ok {
			ts = int64(t)
		}
	}

	if ts == 0 { // Fallback if server time parsing failed
		ts = time.Now().Unix() - 5
	}

	// User documentation says "Nonce (default 0)"
	nonce := int64(0)

	sig, err := c.signer.SignClobAuth(nonce, ts)
	if err != nil {
		return fmt.Errorf("failed to sign auth message: %w", err)
	}

	signerAddr := crypto.PubkeyToAddress(c.signer.privateKey.PublicKey).String()
	c.logger.Infow("Deriving API Credentials", "address", signerAddr, "ts", ts, "nonce", nonce)

	url := fmt.Sprintf("%s/auth/derive-api-key", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Method = "GET"

	req.Header.Set("POLY_ADDRESS", signerAddr)
	req.Header.Set("POLY_SIGNATURE", sig)
	req.Header.Set("POLY_TIMESTAMP", fmt.Sprintf("%d", ts))
	req.Header.Set("POLY_NONCE", fmt.Sprintf("%d", nonce))

	req.Header.Set("POLY_NONCE", fmt.Sprintf("%d", nonce))

	var creds DeriveKeyResponse
	if err := c.do(req, &creds, false); err != nil { // derive uses manual headers
		return fmt.Errorf("failed to derive api key: %w", err)
	}

	c.creds.APIKey = creds.ApiKey
	c.creds.APISecret = creds.Secret
	c.creds.APIPassphrase = creds.Passphrase

	c.logger.Info("Successfully derived API Credentials")
	return nil
}

func (c *ClobClient) GetOrderbook(ctx context.Context, tokenID string) (*Orderbook, error) {
	url := fmt.Sprintf("%s/book?token_id=%s", c.baseURL, tokenID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp Orderbook
	if err := c.do(req, &resp, false); err != nil { // public
		return nil, err
	}

	// Sort Bids: Descending by Price (Highest first)
	sort.Slice(resp.Bids, func(i, j int) bool {
		p1, _ := strconv.ParseFloat(resp.Bids[i].Price, 64)
		p2, _ := strconv.ParseFloat(resp.Bids[j].Price, 64)
		return p1 > p2
	})

	// Sort Asks: Ascending by Price (Lowest first)
	sort.Slice(resp.Asks, func(i, j int) bool {
		p1, _ := strconv.ParseFloat(resp.Asks[i].Price, 64)
		p2, _ := strconv.ParseFloat(resp.Asks[j].Price, 64)
		return p1 < p2
	})

	return &resp, nil
}

// GetOrder fetches a single order.
func (c *ClobClient) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}
	// Endpoint: /data/order/:id (per TS SDK)
	url := fmt.Sprintf("%s/data/order/%s", c.baseURL, orderID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	if err := c.do(req, &resp, true); err != nil { // private
		return nil, err
	}

	return &resp, nil
}

// CreateOrder creates and signs an order, then posts it
func (c *ClobClient) CreateOrder(ctx context.Context, params UserOrderParams, tickSize string, negRisk bool) (*OrderResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	if c.orderBuilder == nil {
		return nil, fmt.Errorf("order builder not initialized")
	}

	opts := BuildOrderOptions{
		TickSize: tickSize,
		NegRisk:  negRisk,
		Maker:    c.creds.FunderAddress,
	}

	signedOrder, err := c.orderBuilder.BuildOrder(params, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build order: %w", err)
	}

	// Create payload
	payload := map[string]interface{}{
		"order":     signedOrder,
		"owner":     c.creds.APIKey,
		"orderType": "GTC",
		"deferExec": false,
	}

	url := fmt.Sprintf("%s/order", c.baseURL)
	body, _ := json.Marshal(payload)

	// Debug log
	c.logger.Infow("Sending order", "payload", string(body))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	if err := c.do(req, &resp, true); err != nil { // private
		return nil, err
	}

	return &resp, nil
}

// CreateMarketOrder creates a market order by calculating price from orderbook and building a FOK order, then posts it
func (c *ClobClient) CreateMarketOrder(ctx context.Context, userMarketOrder UserMarketOrderParams, tickSize string, negRisk bool) (*OrderResponse, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	// If price is not provided, calculate it from orderbook
	if userMarketOrder.Price == nil {
		price, err := c.CalculateMarketPrice(ctx, userMarketOrder.TokenID, userMarketOrder.Side, userMarketOrder.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate market price: %w", err)
		}
		userMarketOrder.Price = &price
	}

	buildOpts := BuildOrderOptions{
		TickSize: NormalizeTickSize(tickSize),
		NegRisk:  negRisk,
		Maker:    c.creds.FunderAddress,
	}

	// Default to FAK if not specified
	if userMarketOrder.OrderType == "" {
		userMarketOrder.OrderType = OrderTypeFAK
	}

	signedOrder, err := c.orderBuilder.BuildMarketOrder(userMarketOrder, buildOpts)
	if err != nil {
		return nil, err
	}

	// Create payload
	payload := map[string]interface{}{
		"order":     signedOrder,
		"owner":     c.creds.APIKey,
		"orderType": userMarketOrder.OrderType,
		"deferExec": false,
	}

	url := fmt.Sprintf("%s/order", c.baseURL)
	body, _ := json.Marshal(payload)

	// Debug log
	c.logger.Infow("Sending market order", "payload", string(body))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	if err := c.do(req, &resp, true); err != nil { // private
		return nil, err
	}

	return &resp, nil
}

// CalculateMarketPrice calculates the worst price needed to fill the amount by traversing the orderbook
func (c *ClobClient) CalculateMarketPrice(ctx context.Context, tokenID string, side Side, amount float64) (float64, error) {
	orderbook, err := c.GetOrderbook(ctx, tokenID)
	if err != nil {
		return 0, err
	}

	// For BUY: We need ASKS (People selling)
	// For SELL: We need BIDS (People buying)
	var levels []OrderSummary
	if side == SideBuy {
		levels = orderbook.Asks
	} else {
		levels = orderbook.Bids
	}

	if len(levels) == 0 {
		return 0, fmt.Errorf("insufficient liquidity: empty orderbook")
	}

	remainingAmount := amount
	worstPrice := 0.0

	for _, level := range levels {
		p, err := strconv.ParseFloat(level.Price, 64)
		if err != nil {
			continue
		}
		s, err := strconv.ParseFloat(level.Size, 64)
		if err != nil {
			continue
		}

		if side == SideBuy {
			// BUY: Amount is USDC (Value)
			// Level Value = Price * Size
			levelValue := p * s

			if levelValue >= remainingAmount {
				// This level covers the rest
				worstPrice = p
				remainingAmount = 0
				break
			} else {
				// Consume this level
				remainingAmount -= levelValue
				worstPrice = p
			}
		} else {
			// SELL: Amount is Tokens (Size)
			// Level Size = Size

			if s >= remainingAmount {
				// This level covers the rest
				worstPrice = p
				remainingAmount = 0
				break
			} else {
				// Consume this level
				remainingAmount -= s
				worstPrice = p
			}
		}
	}

	if remainingAmount > 0.000001 { // Check with small epsilon
		return 0, fmt.Errorf("insufficient liquidity to fill amount %.4f (remaining: %.4f)", amount, remainingAmount)
	}

	return worstPrice, nil
}

// CancelOrder cancels a single order
func (c *ClobClient) CancelOrder(ctx context.Context, orderID string) error {
	if err := c.EnsureAuth(ctx); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	// Use the correct endpoint: DELETE /order (not /order/:id)
	url := fmt.Sprintf("%s/order", c.baseURL)

	// Request body with orderID
	payload := map[string]string{
		"orderID": orderID,
	}

	var response CancelOrderResponse
	if err := c.delete(ctx, url, payload, &response, true); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

// CancelAllOrders cancels all open orders
func (c *ClobClient) CancelAllOrders(ctx context.Context) error {
	if err := c.EnsureAuth(ctx); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	// Use DELETE /cancel-all endpoint
	url := fmt.Sprintf("%s/cancel-all", c.baseURL)

	var response CancelAllResponse
	if err := c.delete(ctx, url, nil, &response, true); err != nil {
		return fmt.Errorf("failed to cancel all orders: %w", err)
	}

	return nil
}

// Helper for HTTP requests
func (c *ClobClient) do(req *http.Request, result interface{}, auth bool) error {
	// Add Headers
	req.Header.Set("Content-Type", "application/json")

	// L2 Authentication using HMAC-SHA256
	if auth && c.creds != nil && c.creds.APIKey != "" {
		// Read request body if present
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body
		}

		// Get current timestamp (seconds) - local time should be close enough for L2
		timestamp := time.Now().Unix()

		// Build signature message: timestamp + method + path + body
		// Note from TS SDK: it signs requestPath which typically excludes query parameters
		// even if the request itself has them.
		requestPath := req.URL.Path
		// if req.URL.RawQuery != "" {
		// 	requestPath += "?" + req.URL.RawQuery
		// }
		message := fmt.Sprintf("%d%s%s%s", timestamp, req.Method, requestPath, string(bodyBytes))

		// Decode the secret from base64 (critical!)
		// Try multiple decoding strategies
		var secretBytes []byte
		var err error

		// Strategy 1: URL-safe base64 with normalization
		normalizedSecret := strings.ReplaceAll(c.creds.APISecret, "-", "+")
		normalizedSecret = strings.ReplaceAll(normalizedSecret, "_", "/")
		if m := len(normalizedSecret) % 4; m != 0 {
			normalizedSecret += strings.Repeat("=", 4-m)
		}
		secretBytes, err = base64.StdEncoding.DecodeString(normalizedSecret)

		// Strategy 2: If strategy 1 failed, try RawURLEncoding (no padding)
		if err != nil {
			secretBytes, err = base64.RawURLEncoding.DecodeString(c.creds.APISecret)
		}

		// Strategy 3: If all else fails, use raw bytes (unlikely but safe fallback)
		if err != nil {
			c.logger.Warnw("Failed to decode API secret as base64, using raw bytes", "error", err)
			secretBytes = []byte(c.creds.APISecret)
		}

		// Compute HMAC-SHA256
		h := hmac.New(sha256.New, secretBytes)
		h.Write([]byte(message))
		signatureBytes := h.Sum(nil)

		// Encode to base64 and make it URL-safe
		signature := base64.StdEncoding.EncodeToString(signatureBytes)
		// Convert to URL-safe: + -> -, / -> _
		signature = strings.ReplaceAll(signature, "+", "-")
		signature = strings.ReplaceAll(signature, "/", "_")

		// Debug logging
		c.logger.Debugw("L2 HMAC Auth", "timestamp", timestamp, "method", req.Method, "path", requestPath, "body_len", len(bodyBytes))

		// Get signer address for POLY_ADDRESS header.
		// Always use EOA address here — the API key is derived from the EOA,
		// so auth headers must match the EOA. The Safe address is only used
		// as the "maker" in order signatures, not for API authentication.
		var address string
		if c.signer != nil {
			address = crypto.PubkeyToAddress(c.signer.privateKey.PublicKey).String()
		}

		// Set L2 auth headers (matching TypeScript SDK)
		if address != "" {
			req.Header.Set("POLY_ADDRESS", address)
		}
		req.Header.Set("POLY_API_KEY", c.creds.APIKey)
		req.Header.Set("POLY_SIGNATURE", signature)
		req.Header.Set("POLY_TIMESTAMP", fmt.Sprintf("%d", timestamp))
		req.Header.Set("POLY_PASSPHRASE", c.creds.APIPassphrase)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Debug log raw response if error
	if res.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		var errResp ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Message != "" {
			return fmt.Errorf("API error: %s (code: %d)", errResp.Message, errResp.Code)
		}

		return fmt.Errorf("API error: status code %d, body: %s", res.StatusCode, bodyString)
	}

	if result != nil {
		return json.NewDecoder(res.Body).Decode(result)
	}
	return nil
}

// Generic HTTP helpers for market.go and other modules

// get performs a GET request with query parameters
func (c *ClobClient) get(ctx context.Context, url string, params map[string]string, result interface{}, auth bool) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	return c.do(req, result, auth)
}

// post performs a POST request with JSON body
func (c *ClobClient) post(ctx context.Context, url string, body interface{}, result interface{}, auth bool) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.do(req, result, auth)
}

// delete performs a DELETE request with optional JSON body
func (c *ClobClient) delete(ctx context.Context, url string, body interface{}, result interface{}, auth bool) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, reqBody)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.do(req, result, auth)
}

// Adapter Interface Helpers

// ToGenericOrderbook converts the Polymarket Orderbook to the generic prediction.Orderbook
func (ob *Orderbook) ToGeneric(marketID string) *prediction.Orderbook {
	// Conversion logic
	// Note: Polymarket OB has prices as strings
	// Implementation required parsing strings to float64
	return &prediction.Orderbook{
		MarketID: marketID,
		TokenID:  ob.AssetID,
		// ... mapping fields
	}
}
