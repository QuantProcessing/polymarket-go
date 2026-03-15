# Polymarket Go SDK

[中文文档](README_zh.md)

> [!WARNING]
> This project is a Go rewrite of [poly-sdk](https://github.com/cyl19970726/poly-sdk) (TypeScript), generated with the assistance of AI. It has **not** been rigorously tested. Use at your own risk in production environments.

A feature-complete Go SDK for the [Polymarket](https://polymarket.com) prediction market platform.

## Architecture

The SDK uses a **dual-layer architecture**:

1. **Service Layer** (`services/`): The **recommended** interface. Aggregates multiple low-level clients into high-level, business-oriented APIs (Trading, Market, Realtime).
2. **Client Layer** (`clients/`): The **low-level protocol** implementation. Directly maps to Polymarket's backend services (CLOB, Gamma, Data API, etc.).

## Installation

```bash
go get github.com/QuantProcessing/polymarket-go
```

## Quick Start

Use the **Root Client** (`polymarket.NewClient`) — it initializes all services automatically.

### Initialization

```go
import (
    "context"
    "go.uber.org/zap"
    polymarket "github.com/QuantProcessing/polymarket-go"
)

func main() {
    config := polymarket.Config{
        // EOA private key for signing
        PrivateKey: "0x...",

        // Optional: Gnosis Safe proxy address
        FunderAddress: "0x...",

        // Optional: CLOB API credentials (for higher rate limits)
        APIKey:        "...",
        APISecret:     "...",
        APIPassphrase: "...",
    }

    logger, _ := zap.NewDevelopment()

    client, err := polymarket.NewClient(context.Background(), config, logger.Sugar())
    if err != nil {
        panic(err)
    }

    // Use client.Trading, client.Market, client.Realtime
}
```

---

## Components

### 1. Trading Service (`TradingService`)

Handles order management, fund operations, and on-chain interactions.
**Dependencies**: `clients/clob`, `clients/ctf`

#### Create Order

```go
import "github.com/QuantProcessing/polymarket-go/clients/clob"

params := clob.UserOrderParams{
    TokenID: "1234...",       // Get from MarketService
    Price:   0.50,
    Size:    100.0,
    Side:    clob.SideBuy,
}

// Automatically handles: balance/approval checks, EIP-712 signing, CLOB submission
resp, err := client.Trading.CreateOrder(ctx, params)
```

#### Cancel Order

```go
// Cancel a single order
client.Trading.CancelOrder(ctx, "order-id-...")

// Cancel all orders
client.Trading.CancelAllOrders(ctx)
```

#### On-Chain Operations (CTF)

```go
// Redeem winning positions (auto-detects EOA vs Gnosis Safe)
txHash, err := client.Trading.Redeem(ctx, conditionID, "YES")
```

---

### 2. Market Service (`MarketService`)

Aggregates Gamma (market data), Data API (historical/stats), and Subgraph (on-chain data).
**Dependencies**: `clients/gamma`, `clients/data`, `clients/subgraph`

#### Get Market Details

```go
// By Condition ID (recommended)
market, err := client.Market.GetMarketByConditionID(ctx, "0x...")

// Get Token IDs for trading
yesTokenID := market.Tokens[0].TokenID
noTokenID  := market.Tokens[1].TokenID
```

#### Search Markets

```go
import "github.com/QuantProcessing/polymarket-go/clients/gamma"

params := gamma.MarketSearchParams{
    Query: "Trump",
    Limit: 10,
}
markets, err := client.Market.GetMarkets(ctx, params)
```

#### Price History

```go
history, err := client.Market.GetPriceHistory(ctx, clob.PriceHistoryFilterParams{
    Market:   "token-id...",
    Interval: "1h",
})
```

---

### 3. Realtime Service (`RealtimeService`)

Manages WebSocket connections with automatic reconnection.
**Dependencies**: `clients/rtds`

Features:
- Separate **Market** (public) and **User** (private orders/fills) channels
- Automatic reconnection

#### Subscribe to Market Data

```go
// 1. Connect (call once at startup)
err := client.Realtime.ConnectAll()

// 2. Subscribe to specific assets
err = client.Realtime.Market.SubscribeMarket([]string{tokenID1, tokenID2})

// 3. Consume data in a goroutine
go func() {
    for msg := range client.Realtime.Market.Messages() {
        // msg is []byte, parse as clob.OrderbookUpdate etc.
    }
}()
```

#### Subscribe to Chainlink Prices

```go
client.Realtime.Market.SubscribeCryptoPrices([]string{"ETH", "BTC"})
```

---

## Advanced: Direct Client Access

For finer-grained control, access low-level clients via the Root Client:

| Client | Description |
| :--- | :--- |
| `client.Clob` | CLOB REST API (`GetServerTime`, `GetAPIKeys`) |
| `client.Ctf` | Smart contract methods (`Split`, `Merge`) |
| `client.Data` | Data API (`GetLeaderboard`) |
| `client.Gamma` | Gamma API (`GetEventBySlug`) |
| `client.Subgraph` | Custom GraphQL queries |

## Configuration

| Field | Description | Required |
| :--- | :--- | :--- |
| `PrivateKey` | EOA wallet private key (Hex). Used for signing. | Yes (for trading) |
| `FunderAddress` | Gnosis Safe proxy address. Enables EIP-1271 signing. | No |
| `APIKey` / `Secret` / `Passphrase` | CLOB API credentials for WebSocket auth and rate limits. | No (recommended) |
| `RPCURL` | Polygon RPC endpoint for on-chain operations (CTF). | Yes (for on-chain) |

## Project Structure

```
polymarket-go/
├── client.go           # Root Client (entry point)
├── services/           # Service Layer
│   ├── trading/        # Trading & accounts
│   ├── market/         # Market data & discovery
│   ├── realtime/       # Real-time streaming
│   ├── ordermanager/   # Order lifecycle management
│   └── arb/            # Arbitrage utilities
└── clients/            # Client Layer (protocol implementations)
    ├── clob/           # CLOB REST/WS
    ├── ctf/            # On-chain contracts
    ├── gamma/          # Market discovery API
    ├── data/           # Data statistics API
    ├── rtds/           # Real-time data service
    └── subgraph/       # GraphQL queries
```

## License

[MIT](LICENSE)
