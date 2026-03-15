# Polymarket Go SDK

这是一个功能完备的 Polymarket Go 语言客户端 SDK。

## 架构概览

本 SDK 采用了 **双层架构** 设计：

1.  **服务层 (Services)**：位于 `services/` 目录。这是**推荐的使用方式**。它聚合了多个底层客户端，提供了面向业务的高级接口（如“交易”、“行情”、“实时数据”）。
2.  **客户端层 (Clients)**：位于 `clients/` 目录。这是**底层协议实现**。它直接对应 Polymarket 的各个后端服务（CLOB, Gamma, Data API 等）。

## 快速开始

推荐使用 **Root Client** (`polymarket.NewClient`)，它会自动初始化所有服务。

### 初始化

```go
import (
    "context"
    "time"
    "go.uber.org/zap"
    "trader/pkg/prediction/polymarket"
)

func main() {
    // 基础配置
    config := polymarket.Config{
        // 交易需要私钥 (EOA)
        PrivateKey: "0x...", 
        
        // 或者是 Gnosis Safe 的代理地址
        FunderAddress: "0x...", 

        // CLOB API 凭证 (可选，用于提升限频或访问专用接口)
        APIKey:        "...",
        APISecret:     "...",
        APIPassphrase: "...",
    }

    logger, _ := zap.NewDevelopment()
    
    // 初始化 SDK
    client, err := polymarket.NewClient(context.Background(), config, logger.Sugar())
    if err != nil {
        panic(err)
    }

    // 现在你可以使用 client.Trading, client.Market, client.Realtime 了
}
```

---

## 组件详解

### 1. 交易服务 (`TradingService`)

**作用**：处理订单管理、资金操作和链上交互。
**底层依赖**：`clients/clob`, `clients/ctf`

**常见场景**：

*   **创建订单 (Create Order)**

```go
import (
    "trader/pkg/prediction/polymarket/clients/clob"
)

// 创建限价单
params := clob.UserOrderParams{
    TokenID: "1234...", // Token ID (通过 MarketService 获取)
    Price:   0.50,      // 价格
    Size:    100.0,     // 数量 (Share数)
    Side:    clob.SideBuy, 
}

// 自动处理：
// 1. 检查余额/授权 (Proxy或EOA)
// 2. 签名 (EIP-712)
// 3. 提交到 CLOB
resp, err := client.Trading.CreateOrder(ctx, params)
```

*   **撤单 (Cancel Order)**

```go
// 单个撤单
client.Trading.CancelOrder(ctx, "order-id-...")

// 撤销所有订单
client.Trading.CancelAllOrders(ctx)
```

*   **链上操作 (CTF)**

```go
// 赎回获胜头寸 (自动判断是 EOA 还是 Gnosis Safe 代理操作)
txHash, err := client.Trading.Redeem(ctx, conditionID, "YES")
```

---

### 2. 行情服务 (`MarketService`)

**作用**：聚合了 Gamma (主要市场数据)、Data API (历史/统计) 和 Subgraph (链上数据)。
**底层依赖**：`clients/gamma`, `clients/data`, `clients/subgraph`

**常见场景**：

*   **获取市场详情**

```go
// 通过 Condition ID (推荐)
market, err := client.Market.GetMarketByConditionID(ctx, "0x...")

// 获取 Token ID 用于交易
yesTokenID := market.Tokens[0].TokenID
noTokenID  := market.Tokens[1].TokenID
```

*   **搜索市场**

```go
import "trader/pkg/prediction/polymarket/clients/gamma"

// 搜索 "Trump" 相关的市场
params := gamma.MarketSearchParams{
    Query: "Trump",
    Limit: 10,
}
markets, err := client.Market.GetMarkets(ctx, params)
```

*   **获取价格历史**

```go
// 获取 K 线数据
history, err := client.Market.GetPriceHistory(ctx, clob.PriceHistoryFilterParams{
    Market: "token-id...",
    Interval: "1h",
})
```

---

### 3. 实时数据服务 (`RealtimeService`)

**作用**：统一管理 WebSocket 连接。
**底层依赖**：`clients/rtds` (Polymarket Real-Time Data Service)

**特点**：
- 自动管理重连。
- 分离 **Market** (公共数据) 和 **User** (私有订单/成交) 通道。

**常见场景**：

*   **订阅市场数据**

```go
// 1. 连接 (建议在程序启动时调用)
err := client.Realtime.ConnectAll()

// 2. 订阅特定资产的价格/Orderbook更新
err := client.Realtime.Market.SubscribeMarket([]string{tokenID1, tokenID2})

// 3. 消费数据 (在独立 goroutine 中)
go func() {
    for msg := range client.Realtime.Market.Messages() {
        // 处理 JSON 消息
        // msg 是 []byte，可以解析为 clob.OrderbookUpdate 等结构
    }
}()
```

*   **订阅 Chainlink 价格**

```go
// 直接订阅底层资产价格 (如 ETH, BTC)
// RTDS 服务特有功能
client.Realtime.Market.SubscribeCryptoPrices([]string{"ETH", "BTC"})
```

---

## 进阶：直接使用底层客户端

如果你需要更精细的控制，可以通过 Root Client 直接访问底层 Clients：

*   `client.Clob`: 直接调用 CLOB REST API (如 `GetServerTime`, `GetAPIKeys`)。
*   `client.Ctf`: 直接调用智能合约方法 (如 `Split`, `Merge` 的底层逻辑)。
*   `client.Data`: 调用 Data API (如 `GetLeaderboard`)。
*   `client.Gamma`: 调用 Gamma API (复杂查询 `GetEventBySlug`)。
*   `client.Subgraph`: 执行自定义 GraphQL 查询。

## 配置项说明

| 字段 | 说明 | 必填 |
| :--- | :--- | :--- |
| `PrivateKey` | EOA 钱包私钥 (Hex)。用于签名交易和订单。 | 是 (交易时) |
| `FunderAddress` | Gnosis Safe 代理合约地址。如果设置，SDK 会自动使用 EIP-1271 签名并作为代理执行。 | 否 |
| `APIKey` / `Secret` / `Passphrase` | CLOB API 凭证。用于 WebSocket 认证和提升 REST 限频。 | 否 (建议填) |
| `RPCURL` | Polygon 节点地址。用于链上交互 (CTF)。 | 是 (链上操作时) |

## 目录结构

```
pkg/prediction/polymarket/
├── client.go           # Root Client (入口)
├── services/           # 服务层
│   ├── trading/        # 交易与账户
│   ├── market/         # 行情与数据
│   └── realtime/       # 实时推送
└── clients/            # 客户端层 (协议实现)
    ├── clob/           # CLOB REST/WS
    ├── ctf/            # 链上合约
    ├── gamma/          # 市场发现 API
    ├── data/           # 数据统计 API
    ├── rtds/           # 实时数据服务
    └── ...
```
