---
name: polymarket-go
description: Use when building Go applications that interact with Polymarket prediction markets, placing orders, fetching market data, or subscribing to real-time price feeds via WebSocket
---

# Polymarket Go SDK

## Overview

Go SDK for Polymarket prediction market. Dual-layer architecture: **Service Layer** (recommended, high-level business APIs) and **Client Layer** (low-level protocol access).

> [!CAUTION]
> AI-rewritten from [poly-sdk](https://github.com/cyl19970726/poly-sdk) (TypeScript). Not rigorously tested — verify behavior before production use.

## When to Use

- Building Go trading bots or analytics for Polymarket
- Fetching prediction market data (prices, orderbooks, events)
- Placing/canceling orders via CLOB API
- Subscribing to real-time WebSocket feeds
- On-chain CTF operations (split, merge, redeem)

**Not for:** TypeScript projects (use [poly-sdk](https://github.com/cyl19970726/poly-sdk) directly).

## Quick Reference

| Component | Import | Purpose |
|---|---|---|
| Root Client | `polymarket "github.com/QuantProcessing/polymarket-go"` | Entry point, initializes all services |
| Trading | `services/trading` | Orders, cancellations, on-chain ops |
| Market | `services/market` | Market data, search, price history |
| Realtime | `services/realtime` | WebSocket streaming |
| OrderManager | `services/ordermanager` | Order lifecycle management |
| CLOB Client | `clients/clob` | Direct CLOB REST/WS API |
| CTF Client | `clients/ctf` | Smart contract interactions |
| Gamma Client | `clients/gamma` | Market discovery API |
| Data Client | `clients/data` | Historical statistics |

## Core Pattern

```go
import (
    "context"
    "go.uber.org/zap"
    polymarket "github.com/QuantProcessing/polymarket-go"
    "github.com/QuantProcessing/polymarket-go/clients/clob"
)

// 1. Initialize
config := polymarket.Config{
    PrivateKey:    "0x...",
    APIKey:        "...",
    APISecret:     "...",
    APIPassphrase: "...",
}
logger, _ := zap.NewDevelopment()
client, _ := polymarket.NewClient(context.Background(), config, logger.Sugar())

// 2. Fetch market
market, _ := client.Market.GetMarketByConditionID(ctx, "0x...")

// 3. Place order
resp, _ := client.Trading.CreateOrder(ctx, clob.UserOrderParams{
    TokenID: market.Tokens[0].TokenID,
    Price:   0.50,
    Size:    100.0,
    Side:    clob.SideBuy,
})

// 4. Stream real-time data
client.Realtime.ConnectAll()
client.Realtime.Market.SubscribeMarket([]string{tokenID})
go func() {
    for msg := range client.Realtime.Market.Messages() {
        // process []byte message
    }
}()
```

## Configuration

| Field | Required | Description |
|---|---|---|
| `PrivateKey` | Yes (trading) | EOA private key (Hex) for signing |
| `FunderAddress` | No | Gnosis Safe proxy for EIP-1271 signing |
| `APIKey/Secret/Passphrase` | No (recommended) | CLOB credentials for auth & rate limits |
| `RPCURL` | Yes (on-chain) | Polygon RPC for CTF operations |
| `ChainID` | Yes (on-chain) | Chain ID (137 for Polygon mainnet) |

## Common Mistakes

| Mistake | Fix |
|---|---|
| Missing `RPCURL` for on-chain ops | Set `RPCURL` in config; without it CTF client is nil |
| Not calling `ConnectAll()` before subscribe | WebSocket must connect before subscribing to channels |
| Blocking on `Messages()` channel | Always consume in a separate goroutine |
| Using old import paths | Use `github.com/QuantProcessing/polymarket-go/...` |
