package rtds

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"


	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	RTDSMarketURL   = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	RTDSUserURL     = "wss://ws-subscriptions-clob.polymarket.com/ws/user"
	RTDSLiveDataURL = "wss://ws-live-data.polymarket.com"

	pingInterval = 30 * time.Second
	pongTimeout  = 10 * time.Second
	readTimeout  = 60 * time.Second

	initialReconnectWait = 1 * time.Second
	maxReconnectWait     = 30 * time.Second
	reconnectBackoff     = 2.0
)

// CryptoPricePayload represents crypto price data from Chainlink
type CryptoPricePayload struct {
	Symbol    string  `json:"symbol"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// CryptoPriceUpdate represents full crypto price update message
type CryptoPriceUpdate struct {
	Topic     string             `json:"topic"`
	Type      string             `json:"type"`
	Timestamp int64              `json:"timestamp"`
	Payload   CryptoPricePayload `json:"payload"`
}

// RTDSConfig configuration for RTDS Client
type RTDSConfig struct {
	URL           string
	AutoReconnect bool
}

// RTDSClient handles real-time data subscriptions via WebSocket
type RTDSClient struct {
	url       string
	ctx       context.Context
	cancel    context.CancelFunc
	conn      *websocket.Conn
	mu        sync.Mutex
	messageCh chan []byte
	errCh     chan error
	done      chan struct{}
	log       *zap.SugaredLogger
	handlers  []func([]byte)
	isClosed  bool
	reconnect bool
	dialer    *websocket.Dialer

	// Subscription tracking for auto-recovery
	subscriptions []map[string]interface{}
	// For typed handlers (like prices)
	priceHandlers map[string]func(*CryptoPricePayload) error
}

// NewRTDSClient creates a new Real-Time Data Service client
func NewRTDSClient(ctx context.Context, config RTDSConfig) *RTDSClient {
	if config.URL == "" {
		config.URL = RTDSMarketURL
	}

	ctx, cancel := context.WithCancel(ctx)

	return &RTDSClient{
		url:       config.URL,
		ctx:       ctx,
		cancel:    cancel,
		messageCh: make(chan []byte, 100),
		errCh:     make(chan error, 10),
		done:      make(chan struct{}),
		log:       zap.NewNop().Sugar(),
		reconnect: config.AutoReconnect,
		dialer: &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			Proxy:            http.ProxyFromEnvironment,
		},
		subscriptions: make([]map[string]interface{}, 0),
		priceHandlers: make(map[string]func(*CryptoPricePayload) error),
	}
}

// Connect establishes the WebSocket connection
func (c *RTDSClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	c.log.Info("Connecting to RTDS...", "url", c.url)

	conn, _, err := c.dialer.DialContext(c.ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to RTDS: %w", err)
	}

	c.conn = conn
	c.isClosed = false
	c.done = make(chan struct{})

	// Set ping handler
	c.conn.SetPingHandler(func(appData string) error {
		c.log.Debug("Received Ping", "data", appData)
		return c.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(pongTimeout))
	})

	// Set pong handler
	c.conn.SetPongHandler(func(appData string) error {
		c.log.Debug("Received Pong")
		return c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	})

	// Start read loop
	go c.readLoop()
	// Start ping loop
	go c.pingLoop()

	c.log.Info("Connected to RTDS")
	return nil
}

// SubscribeMarket subscribes to market events (book, price_change, etc.)
func (c *RTDSClient) SubscribeMarket(assetIDs []string) error {
	msg := map[string]interface{}{
		"type":       "market",
		"assets_ids": assetIDs,
	}

	if err := c.sendJSON(msg); err != nil {
		return err
	}

	c.mu.Lock()
	c.subscriptions = append(c.subscriptions, msg)
	c.mu.Unlock()
	return nil
}

// SubscribeUser subscribes to user events (requires auth)
func (c *RTDSClient) SubscribeUser(apiKey, secret, passphrase string, markets []string) error {
	msg := map[string]interface{}{
		"type": "user",
		"auth": map[string]string{
			"apiKey":     apiKey,
			"secret":     secret,
			"passphrase": passphrase,
		},
	}
	if len(markets) > 0 {
		msg["markets"] = markets
	}

	if err := c.sendJSON(msg); err != nil {
		return err
	}

	c.mu.Lock()
	c.subscriptions = append(c.subscriptions, msg)
	c.mu.Unlock()
	return nil
}

// SubscribeCryptoPrices subscribes to Binance crypto prices
func (c *RTDSClient) SubscribeCryptoPrices(symbols []string) error {
	for _, symbol := range symbols {
		filter := map[string]string{"symbol": symbol}
		filterBytes, _ := json.Marshal(filter)

		msg := map[string]interface{}{
			"action": "subscribe",
			"subscriptions": []map[string]interface{}{
				{
					"topic":   "crypto_prices",
					"type":    "*",
					"filters": string(filterBytes),
				},
			},
		}
		if err := c.sendJSON(msg); err != nil {
			return err
		}

		c.mu.Lock()
		c.subscriptions = append(c.subscriptions, msg)
		c.mu.Unlock()
	}
	return nil
}

// SubscribeCryptoPricesChainlink subscribes to Chainlink crypto prices
func (c *RTDSClient) SubscribeCryptoPricesChainlink(symbols []string) error {
	for _, symbol := range symbols {
		filter := map[string]string{"symbol": symbol}
		filterBytes, _ := json.Marshal(filter)

		msg := map[string]interface{}{
			"action": "subscribe",
			"subscriptions": []map[string]interface{}{
				{
					"topic":   "crypto_prices_chainlink",
					"type":    "*",
					"filters": string(filterBytes),
				},
			},
		}
		if err := c.sendJSON(msg); err != nil {
			return err
		}

		c.mu.Lock()
		c.subscriptions = append(c.subscriptions, msg)
		c.mu.Unlock()
	}
	return nil
}

// Messages returns the read-only channel for incoming messages
func (c *RTDSClient) Messages() <-chan []byte {
	return c.messageCh
}

// Errors returns the read-only channel for errors
func (c *RTDSClient) Errors() <-chan error {
	return c.errCh
}

// Close closes the connection permanently (disables reconnect, cancels context).
func (c *RTDSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reconnect = false // Disable reconnect on manual close
	c.closeInternal()
}

// ResetConnection closes the current WebSocket connection and clears all subscriptions,
// but preserves the context so Connect() can re-establish a fresh connection.
// Use this between market cycles to cleanly switch subscriptions.
func (c *RTDSClient) ResetConnection() {
	c.mu.Lock()

	// Close underlying WS without cancelling context
	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}

	// Mark as closed so readLoop's deferred cleanup won't auto-reconnect
	if !c.isClosed {
		c.isClosed = true
		close(c.done)
	}

	// Clear stored subscriptions — next Connect+Subscribe will be fresh
	c.subscriptions = c.subscriptions[:0]

	c.mu.Unlock()

	// Wait briefly for readLoop/pingLoop goroutines to fully exit
	time.Sleep(200 * time.Millisecond)

	// Prepare for fresh Connect() by resetting closed state
	c.mu.Lock()
	c.isClosed = false
	c.done = make(chan struct{})
	c.mu.Unlock()

	// Drain any stale messages from the channel
	for {
		select {
		case <-c.messageCh:
		default:
			return
		}
	}
}

func (c *RTDSClient) closeInternal() {
	if c.isClosed {
		return
	}
	c.isClosed = true
	c.cancel()

	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}

	close(c.done)
}

func (c *RTDSClient) sendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}

	return c.conn.WriteJSON(v)
}

func (c *RTDSClient) readLoop() {
	defer func() {
		c.mu.Lock()
		if !c.isClosed && c.reconnect {
			c.mu.Unlock()
			c.handleReconnect()
		} else {
			c.closeInternal()
			c.mu.Unlock()
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// ReadMessage is a blocking call
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					c.log.Error("RTDS read error", "error", err)
					select {
					case c.errCh <- err:
					default:
					}
				}
				return
			}

			// Non-blocking send to message channel
			select {
			case c.messageCh <- message:
			default:
				// c.log.Warn("RTDS message channel full, dropping message")
			}
		}
	}
}

func (c *RTDSClient) pingLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.conn != nil {
				// WriteControl is thread-safe, but we lock to ensure conn is not nil
				err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
				if err != nil {
					c.log.Error("Failed to send ping", "error", err)
					c.mu.Unlock()
					return // Trigger reconnect logic in readLoop (if it detects error) or let readLoop handle timeout
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *RTDSClient) handleReconnect() {
	c.log.Info("Attempting to reconnect to RTDS...")

	retryWait := initialReconnectWait
	c.mu.Lock()
	c.conn = nil // ensure connection is reset
	c.mu.Unlock()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		time.Sleep(retryWait)

		// Try to connect
		conn, _, err := c.dialer.DialContext(c.ctx, c.url, nil)
		if err != nil {
			c.log.Warnw("Reconnection failed", "wait", retryWait, "error", err)
			retryWait = time.Duration(float64(retryWait) * reconnectBackoff)
			if retryWait > maxReconnectWait {
				retryWait = maxReconnectWait
			}
			continue
		}

		// Reconnection successful
		c.mu.Lock()
		c.conn = conn
		c.isClosed = false
		c.done = make(chan struct{})
		// Re-subscribe
		subs := make([]map[string]interface{}, len(c.subscriptions))
		copy(subs, c.subscriptions)
		c.mu.Unlock()

		c.log.Info("Reconnected to RTDS, resubscribing...")

		for _, sub := range subs {
			if err := c.sendJSON(sub); err != nil {
				c.log.Error("Failed to resubscribe", "sub", sub, "error", err)
			}
		}

		// Restart loops
		go c.readLoop()
		go c.pingLoop()

		return
	}
}
