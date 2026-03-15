# Polymarket Go SDK

[English](README.md)

> [!WARNING]
> 本项目是对 [poly-sdk](https://github.com/cyl19970726/poly-sdk)（TypeScript）的 Go 语言重写版本，借助 AI 生成。**尚未经过严格测试**，请在生产环境中谨慎使用。

这是一个功能完备的 [Polymarket](https://polymarket.com) 预测市场 Go 语言 SDK。

## 架构概览

本 SDK 采用 **双层架构** 设计：

1. **服务层 (Services)**：位于 `services/` 目录。这是**推荐的使用方式**。它聚合了多个底层客户端，提供了面向业务的高级接口（如"交易"、"行情"、"实时数据"）。
2. **客户端层 (Clients)**：位于 `clients/` 目录。这是**底层协议实现**。它直接对应 Polymarket 的各个后端服务（CLOB, Gamma, Data API 等）。

## 安装

```bash
go get github.com/QuantProcessing/polymarket-go
```

## 快速开始

推荐使用 **Root Client** (`polymarket.NewClient`)，它会自动初始化所有服务。

### 初始化

```go
import (
    "context"
    "go.uber.org/zap"
    polymarket "github.com/QuantProcessing/polymarket-go"
)

func main() {
    config := polymarket.Config{
        // EOA 钱包私钥，用于签名
        PrivateKey: "0x...",

        // 可选：Gnosis Safe 代理合约地址
        FunderAddress: "0x...",

        // 可选：CLOB API 凭证（用于提升限频）
        APIKey:        "...",
        APISecret:     "...",
        APIPassphrase: "...",
    }

    logger, _ := zap.NewDevelopment()

    client, err := polymarket.NewClient(context.Background(), config, logger.Sugar())
    if err != nil {
        panic(err)
    }

    // 使用 client.Trading, client.Market, client.Realtime
}
```

---

## 组件详解

### 1. 交易服务 (`TradingService`)

**作用**：处理订单管理、资金操作和链上交互。
**底层依赖**：`clients/clob`, `clients/ctf`

#### 创建订单

```go
import "github.com/QuantProcessing/polymarket-go/clients/clob"

params := clob.UserOrderParams{
    TokenID: "1234...", // 通过 MarketService 获取
    Price:   0.50,
    Size:    100.0,
    Side:    clob.SideBuy,
}

// 自动处理：余额/授权检查、EIP-712 签名、提交到 CLOB
resp, err := client.Trading.CreateOrder(ctx, params)
```

#### 撤单

```go
// 单个撤单
client.Trading.CancelOrder(ctx, "order-id-...")

// 撤销所有订单
client.Trading.CancelAllOrders(ctx)
```

#### 链上操作 (CTF)

```go
// 赎回获胜头寸（自动判断 EOA / Gnosis Safe）
txHash, err := client.Trading.Redeem(ctx, conditionID, "YES")
```

---

### 2. 行情服务 (`MarketService`)

**作用**：聚合 Gamma（市场数据）、Data API（历史/统计）和 Subgraph（链上数据）。
**底层依赖**：`clients/gamma`, `clients/data`, `clients/subgraph`

#### 获取市场详情

```go
// 通过 Condition ID（推荐）
market, err := client.Market.GetMarketByConditionID(ctx, "0x...")

// 获取 Token ID 用于交易
yesTokenID := market.Tokens[0].TokenID
noTokenID  := market.Tokens[1].TokenID
```

#### 搜索市场

```go
import "github.com/QuantProcessing/polymarket-go/clients/gamma"

params := gamma.MarketSearchParams{
    Query: "Trump",
    Limit: 10,
}
markets, err := client.Market.GetMarkets(ctx, params)
```

#### 获取价格历史

```go
history, err := client.Market.GetPriceHistory(ctx, clob.PriceHistoryFilterParams{
    Market:   "token-id...",
    Interval: "1h",
})
```

---

### 3. 实时数据服务 (`RealtimeService`)

**作用**：统一管理 WebSocket 连接，支持自动重连。
**底层依赖**：`clients/rtds`

特点：
- 分离 **Market**（公共数据）和 **User**（私有订单/成交）通道
- 自动管理重连

#### 订阅市场数据

```go
// 1. 连接（建议在程序启动时调用）
err := client.Realtime.ConnectAll()

// 2. 订阅特定资产
err = client.Realtime.Market.SubscribeMarket([]string{tokenID1, tokenID2})

// 3. 消费数据（在独立 goroutine 中）
go func() {
    for msg := range client.Realtime.Market.Messages() {
        // msg 是 []byte，可以解析为 clob.OrderbookUpdate 等结构
    }
}()
```

#### 订阅 Chainlink 价格

```go
client.Realtime.Market.SubscribeCryptoPrices([]string{"ETH", "BTC"})
```

---

## 进阶：直接使用底层客户端

如果你需要更精细的控制，可以通过 Root Client 直接访问底层 Clients：

| 客户端 | 说明 |
| :--- | :--- |
| `client.Clob` | CLOB REST API（`GetServerTime`, `GetAPIKeys`） |
| `client.Ctf` | 智能合约方法（`Split`, `Merge`） |
| `client.Data` | Data API（`GetLeaderboard`） |
| `client.Gamma` | Gamma API（`GetEventBySlug`） |
| `client.Subgraph` | 自定义 GraphQL 查询 |

## 配置项说明

| 字段 | 说明 | 必填 |
| :--- | :--- | :--- |
| `PrivateKey` | EOA 钱包私钥 (Hex)。用于签名交易和订单。 | 是（交易时） |
| `FunderAddress` | Gnosis Safe 代理合约地址。设置后 SDK 自动使用 EIP-1271 签名。 | 否 |
| `APIKey` / `Secret` / `Passphrase` | CLOB API 凭证。用于 WebSocket 认证和提升 REST 限频。 | 否（建议填） |
| `RPCURL` | Polygon 节点地址。用于链上交互 (CTF)。 | 是（链上操作时） |

## 环境变量

将 `.env.example` 复制为 `.env` 并填入你的配置：

```bash
cp .env.example .env
```

SDK 及集成测试使用以下环境变量：

| 变量 | 说明 | 必填 |
| :--- | :--- | :--- |
| `POLY_PRIVATE_KEY` | EOA 钱包私钥（Hex，含 `0x` 前缀） | 是（交易时） |
| `POLY_FUNDER_ADDR` | Gnosis Safe 代理地址 | 否 |
| `CLOB_API_KEY` | CLOB API Key | 否（建议填） |
| `CLOB_SECRET` | CLOB API Secret | 否（建议填） |
| `CLOB_PASSPHRASE` | CLOB API Passphrase | 否（建议填） |
| `POLY_RPC_URL` | Polygon RPC 节点地址 | 是（链上操作时） |
| `PROXY` | HTTP/SOCKS5 代理地址（Data API 客户端使用） | 否 |
| `ENABLE_SPEND_TESTS` | 设为 `true` 启用消耗真实资金的测试 | 否 |

## 目录结构

```
polymarket-go/
├── client.go           # Root Client（入口）
├── services/           # 服务层
│   ├── trading/        # 交易与账户
│   ├── market/         # 行情与数据
│   ├── realtime/       # 实时推送
│   ├── ordermanager/   # 订单生命周期管理
│   └── arb/            # 套利工具
└── clients/            # 客户端层（协议实现）
    ├── clob/           # CLOB REST/WS
    ├── ctf/            # 链上合约
    ├── gamma/          # 市场发现 API
    ├── data/           # 数据统计 API
    ├── rtds/           # 实时数据服务
    └── subgraph/       # GraphQL 查询
```

## 许可证

[MIT](LICENSE)
