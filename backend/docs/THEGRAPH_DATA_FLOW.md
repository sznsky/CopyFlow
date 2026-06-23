# The Graph 数据拉取逻辑梳理

## 一、整体架构

```
Worker 启动
    ↓
Service 初始化（V2 Client + V3 Client）
    ↓
并行拉取 V2 和 V3 数据
    ↓
解析 + 过滤 + 标准化
    ↓
存入数据库（upsert）
    ↓
记录同步日志
```

---

## 二、核心文件与职责

| 文件 | 职责 | 关键功能 |
|------|------|---------|
| `pkg/thegraph/client.go` | GraphQL 客户端 | HTTP 请求封装、错误处理 |
| `pkg/thegraph/stablecoins.go` | 业务规则 | 稳定币识别、方向判断 |
| `pkg/thegraph/uniswap.go` | V2/V3 查询 | GraphQL 查询、分页、解析 |
| `internal/smartmoney/sync.go` | 同步逻辑 | 调度 V2/V3、过滤、入库 |
| `cmd/smartmoney-worker/main.go` | 入口 | 启动任务、定时调度 |

---

## 三、详细流程拆解

### 阶段 1：初始化（Worker 启动时）

```go
// cmd/smartmoney-worker/main.go

1. 加载配置
   cfg := config.Load()
   ↓
2. 创建 Service
   service := smartmoney.NewService(
       store,
       cfg.TheGraph.UniswapV2Endpoint,  // V2 子图 URL
       cfg.TheGraph.UniswapV3Endpoint,  // V3 子图 URL
       cfg.TheGraph.APIKey,
       cfg.SmartMoney.ChainID,
       cfg.SmartMoney.BatchSize,        // 分页大小，默认 1000
   )
   ↓
3. Service 内部创建两个 GraphQL 客户端
   v2Client := thegraph.NewClient(v2Endpoint, apiKey)
   v3Client := thegraph.NewClient(v3Endpoint, apiKey)
```

**关键点：**
- V2 和 V3 是**独立的子图**，需要两个 Client
- 每个 Client 有独立的 endpoint 和 60 秒超时

---

### 阶段 2：触发同步（定时或手动）

```go
// Worker 有两个任务并行：

1. 历史数据同步（后台，只运行一次）
   runHistoricalSync()
   ├─ 拉取最近 180 天数据
   ├─ 分批：每批 30 天
   └─ 调用 SyncTradesFromTheGraph(startTime, endTime, minAmountUSD, "historical")

2. 增量同步（定时，每天一次）
   runIncrementalCycle()
   ├─ 拉取昨天的数据
   └─ 调用 SyncTradesFromTheGraph(startTime, endTime, minAmountUSD, "incremental")
```

**参数说明：**
- `startTime` / `endTime`：查询时间范围（UTC 时间戳）
- `minAmountUSD`：最小交易金额（如 10000 USD）
- `syncType`：同步类型（historical / incremental / manual）

---

### 阶段 3：同步主流程（SyncTradesFromTheGraph）

```go
// internal/smartmoney/sync.go

func SyncTradesFromTheGraph(startTime, endTime, minAmountUSD, syncType) {
    
    // 3.1 并行拉取 V2 和 V3
    ├─ v2Inserted, v2Updated, v2Skipped := syncV2Swaps()
    └─ v3Inserted, v3Updated, v3Skipped := syncV3Swaps()
    
    // 3.2 汇总结果
    totalInserted = v2Inserted + v3Inserted
    totalUpdated  = v2Updated  + v3Updated
    totalSkipped  = v2Skipped  + v3Skipped
    
    // 3.3 保存同步日志到 sync_log 表
    syncLog := &model.SyncLog{...}
    store.DB().Create(syncLog)
}
```

**流程特点：**
- V2 和 V3 **顺序执行**（不是 goroutine 并行）
- 即使一个失败，另一个继续
- 最后统一记录日志

---

### 阶段 4：V2 数据拉取（syncV2Swaps）

```go
// 4.1 调用 The Graph API
results := v2Client.FetchUniswapV2Swaps(ctx, startTime, endTime, minAmountUSD, batchSize)
    ↓
// 4.2 FetchUniswapV2Swaps 内部逻辑（pkg/thegraph/uniswap.go）
for {
    // 构造 GraphQL 查询参数
    vars := {
        "first":        1000,         // 每批 1000 条
        "skip":         skip,          // 分页偏移
        "timestampGte": startTime,
        "timestampLt":  endTime,
        "minAmountUSD": "10000.0",
    }
    
    // 执行 GraphQL 查询
    client.Query(ctx, UniswapV2SwapsQuery, vars, &response)
    
    // 累积结果
    allResults.append(response)
    
    // 判断是否继续分页
    if len(response.Swaps) < 1000 {
        break  // 已拉完
    }
    
    skip += 1000
    
    // 限制：skip 不能超过 5000
    if skip >= 5000 {
        return error("reached skip limit")
    }
}
```

**V2 分页策略：**
- 使用 `skip` 偏移分页
- **限制：skip 最大 5000**（The Graph 限制）
- 如果数据量 > 5000，需要缩短时间范围

**GraphQL 查询模板：**
```graphql
query GetSwaps($first: Int!, $skip: Int!, $timestampGte: Int!, ...) {
  swaps(
    first: $first
    skip: $skip
    orderBy: timestamp
    orderDirection: asc
    where: {
      timestamp_gte: $timestampGte
      timestamp_lt: $timestampLt
      amountUSD_gte: $minAmountUSD
    }
  ) {
    id
    transaction { id, timestamp, blockNumber }
    pair {
      id
      token0 { id, symbol }
      token1 { id, symbol }
    }
    to              # V2 的钱包地址
    amount0In       # Token0 输入量
    amount1In       # Token1 输入量
    amount0Out      # Token0 输出量
    amount1Out      # Token1 输出量
    amountUSD       # 交易金额（USD）
  }
}
```

---

### 阶段 5：V3 数据拉取（syncV3Swaps）

```go
// 5.1 调用 The Graph API
results := v3Client.FetchUniswapV3Swaps(ctx, startTime, endTime, minAmountUSD, batchSize)
    ↓
// 5.2 FetchUniswapV3Swaps 内部逻辑
lastID := ""  // 游标

for {
    vars := {
        "first":         1000,
        "timestamp_gte": startTime,
        "timestamp_lt":  endTime,
        "minAmountUSD":  "10000.0",
        "lastID":        lastID,  // 游标分页
    }
    
    client.Query(ctx, UniswapV3SwapsQuery, vars, &response)
    
    allResults.append(response)
    
    // 更新游标
    lastID = response.Swaps[len-1].ID
    
    if len(response.Swaps) < 1000 {
        break
    }
}
```

**V3 分页策略：**
- 使用 `id_gt` 游标分页（更高效）
- **无 5000 限制**，可以拉取任意数量
- 每次用上一批最后一条的 ID 作为下一批起点

**GraphQL 查询模板：**
```graphql
query GetSwaps($first: Int!, $timestamp_gte: Int!, $minAmountUSD: String!, $lastID: String!) {
  swaps(
    first: $first
    orderBy: timestamp
    orderDirection: asc
    where: {
      timestamp_gte: $timestamp_gte
      timestamp_lt: $timestamp_lt
      amountUSD_gte: $minAmountUSD
      id_gt: $lastID           # 游标分页
    }
  ) {
    id
    transaction { id, timestamp, blockNumber }
    pool {
      id
      token0 { id, symbol }
      token1 { id, symbol }
    }
    origin            # V3 的钱包地址（注意：V2 是 `to`）
    amount0           # Token0 变化量（负数=输入，正数=输出）
    amount1           # Token1 变化量
    amountUSD
  }
}
```

---

### 阶段 6：数据解析与标准化

```go
// 6.1 遍历每个 Swap 事件
for _, swap := range batch.Swaps {
    
    // 6.2 解析为标准化结构
    parsed := ParseV2Swap(swap)  // 或 ParseV3Swap(swap)
    
    // ParsedSwap 结构（统一格式）：
    {
        TxHash:          "0xabc...",
        BlockNumber:     12345678,
        Timestamp:       time.Time,
        DEXName:         "uniswap",
        DEXVersion:      "v2" / "v3",
        PoolAddress:     "0xpool...",
        WalletAddress:   "0xwallet...",
        Token0:          "0xtoken0...",
        Token1:          "0xtoken1...",
        Token0Symbol:    "USDT",
        Token1Symbol:    "WETH",
        TokenIn:         "0xtoken0...",  // 根据金额正负判断
        TokenOut:        "0xtoken1...",  // 根据金额正负判断
        TokenInSymbol:   "USDT",
        TokenOutSymbol:  "WETH",
        AmountIn:        "1000.5",
        AmountOut:       "0.5",
        AmountUSD:       "1500.0",
    }
}
```

**V2 vs V3 解析差异：**

| 字段 | V2 | V3 |
|------|----|----|
| 钱包地址 | `to` | `origin` |
| 池子 | `pair.id` | `pool.id` |
| 金额判断 | `amount0In/Out` > 0 判断方向 | `amount0/1` 负数=输入，正数=输出 |

**示例：**
```
V2 Swap:
  amount0In: 1000.0   → TokenIn = Token0
  amount1Out: 0.5     → TokenOut = Token1

V3 Swap:
  amount0: -1000.0    → TokenIn = Token0（负数）
  amount1: 0.5        → TokenOut = Token1（正数）
```

---

### 阶段 7：业务过滤（shouldSkipSwap）

```go
// 7.1 过滤稳定币互换
if IsStablecoinSwap(tokenIn, tokenOut) {
    skip  // USDT → USDC、PAXG → XAUT 等
}

// 7.2 必须能判断买入/卖出方向
isBuy, ok := ClassifyDirection(tokenIn, tokenOut)
if !ok {
    skip  // 既不是买入也不是卖出（如两个非稳定币互换）
}

// 7.3 钱包地址有效性
if walletAddress == "" || walletAddress == "0x0000..." {
    skip
}
```

**方向判断逻辑（ClassifyDirection）：**
```go
// 报价资产 = 稳定币（USDT/USDC/DAI...）+ WETH

if (tokenIn 是报价资产) && (tokenOut 不是报价资产) {
    → 买入   // 用 USDT 买 SHIB
}

if (tokenIn 不是报价资产) && (tokenOut 是报价资产) {
    → 卖出   // 卖 SHIB 换 USDT
}

else {
    → 无法判断（跳过）
}
```

---

### 阶段 8：转换为数据库模型

```go
// 8.1 创建 WalletTrade 模型
trade := &model.WalletTrade{
    ChainID:       1,                           // Ethereum
    TxHash:        strings.ToLower(parsed.TxHash),
    BlockNumber:   parsed.BlockNumber,
    BlockTime:     parsed.Timestamp,
    DEXName:       "uniswap",
    DEXVersion:    "v2" / "v3",
    PoolAddress:   strings.ToLower(parsed.PoolAddress),
    WalletAddress: strings.ToLower(parsed.WalletAddress),
    TokenIn:       strings.ToLower(parsed.TokenIn),
    TokenOut:      strings.ToLower(parsed.TokenOut),
    TokenInSymbol: &parsed.TokenInSymbol,
    TokenOutSymbol: &parsed.TokenOutSymbol,
    AmountIn:      decimal.NewFromString(parsed.AmountIn),
    AmountOut:     decimal.NewFromString(parsed.AmountOut),
    AmountUSD:     decimal.NewFromString(parsed.AmountUSD),
    IsBuy:         isBuy,
    // PnlUSD / PnlPercent 此时为 NULL，后续由 FIFO 匹配计算
}

// 8.2 所有地址统一转小写存储
//     便于后续查询和匹配
```

---

### 阶段 9：数据库 Upsert

```go
// 9.1 检查是否已存在（唯一键）
var existing model.WalletTrade
err := store.DB().Where(
    "tx_hash = ? AND wallet_address = ? AND token_out = ?",
    trade.TxHash,
    trade.WalletAddress,
    trade.TokenOut,
).First(&existing).Error

if err != nil {
    // 9.2 不存在 → 插入
    store.DB().Create(trade)
    inserted++
} else {
    // 9.3 已存在 → 更新（但保留已计算的 PNL）
    store.DB().Model(&existing).
        Omit("PnlUSD", "PnlPercent", "HoldingDurationHours").
        Updates(trade)
    updated++
}
```

**唯一键逻辑：**
- `tx_hash` + `wallet_address` + `token_out`
- 同一笔交易可能有多个钱包参与（如聚合交易）
- 同一钱包在同一交易中可能操作多个代币

**为什么 Omit PNL 字段？**
- PNL 是后续通过 FIFO 算法计算的
- 如果已有 PNL，说明已经匹配过买入/卖出
- 重新同步时不应该覆盖已算好的 PNL

---

### 阶段 10：记录同步日志

```go
// 10.1 保存到 sync_log 表
syncLog := &model.SyncLog{
    Source:          "thegraph",
    SyncType:        "incremental" / "historical",
    ChainID:         1,
    StartTime:       startTime,
    EndTime:         endTime,
    RecordsInserted: totalInserted,
    RecordsUpdated:  totalUpdated,
    RecordsSkipped:  totalSkipped,
    Status:          "success",
    CompletedAt:     time.Now(),
}
store.DB().Create(syncLog)
```

**可查询同步历史：**
```sql
SELECT * FROM sync_log 
ORDER BY created_at DESC 
LIMIT 10;
```

---

## 四、关键技术细节

### 1. GraphQL 客户端封装

```go
// pkg/thegraph/client.go

func (c *Client) Query(ctx, query, variables, result) error {
    // 1. 构造 JSON 请求体
    reqBody := GraphQLRequest{
        Query:     query,
        Variables: variables,
    }
    
    // 2. 发送 HTTP POST
    req := POST(endpoint, jsonData)
    req.Header.Set("Authorization", "Bearer " + apiKey)
    
    // 3. 解析响应
    resp := httpClient.Do(req)
    body := readAll(resp.Body)
    
    // 4. 检查 GraphQL 错误
    if gqlResp.Errors != nil {
        return error
    }
    
    // 5. 反序列化 data 到 result
    json.Unmarshal(gqlResp.Data, result)
}
```

**特点：**
- 60 秒超时
- 自动处理 HTTP 和 GraphQL 双层错误
- 泛型结果反序列化

---

### 2. 分页策略对比

| 特性 | V2 (skip) | V3 (id_gt 游标) |
|------|-----------|----------------|
| 查询 | `skip: 1000` | `id_gt: "0x123..."` |
| 限制 | 最多 5000 条 | 无限制 |
| 性能 | skip 越大越慢 | 始终高效 |
| 使用场景 | 小数据量（< 5000） | 大数据量 |

**为什么 V2 有限制？**
The Graph 为了防止滥用，限制了 skip 参数的最大值。

**解决方案（V2 超 5000）：**
- 缩短时间范围（如 30 天 → 7 天）
- 提高 `minAmountUSD`（减少结果数量）

---

### 3. 稳定币识别

```go
// pkg/thegraph/stablecoins.go

EthereumStablecoins = {
    "0xdac17f958d2ee523a2206206994597c13d831ec7": {}, // USDT
    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": {}, // USDC
    "0x6b175474e89094c44da98b954eedeac495271d0f": {}, // DAI
    // ... 13 种稳定币 + 黄金代币
}

// 过滤逻辑
if IsStablecoin(tokenIn) && IsStablecoin(tokenOut) {
    skip  // USDT ↔ USDC 无分析价值
}
```

---

### 4. 地址规范化

```go
// 统一转小写
trade.WalletAddress = strings.ToLower(walletAddress)
trade.TokenIn       = strings.ToLower(tokenIn)

// 原因：
// - The Graph 返回的地址可能有大小写（EIP-55 checksum）
// - 数据库查询需要统一格式
// - 避免 "0xAbc..." 和 "0xabc..." 被当作两个地址
```

---

## 五、数据流示例

### 完整示例：一笔 V3 Swap

```
1. The Graph 返回：
{
  "id": "0x123...",
  "transaction": {
    "id": "0xabc...",
    "timestamp": "1718892000",
    "blockNumber": "12345678"
  },
  "pool": {
    "id": "0xpool123...",
    "token0": { "id": "0xusdt...", "symbol": "USDT" },
    "token1": { "id": "0xshib...", "symbol": "SHIB" }
  },
  "origin": "0xwallet456...",
  "amount0": "-1000.0",       // USDT 输入（负数）
  "amount1": "5000000.0",     // SHIB 输出（正数）
  "amountUSD": "1000.0"
}

2. 解析为 ParsedSwap：
{
  TxHash: "0xabc...",
  WalletAddress: "0xwallet456...",
  TokenIn: "0xusdt...",       // amount0 < 0
  TokenOut: "0xshib...",      // amount1 > 0
  TokenInSymbol: "USDT",
  TokenOutSymbol: "SHIB",
  AmountIn: "1000.0",
  AmountOut: "5000000.0",
  AmountUSD: "1000.0"
}

3. 方向判断：
IsQuoteToken("0xusdt...") = true    // USDT 是报价资产
IsQuoteToken("0xshib...") = false   // SHIB 不是
→ ClassifyDirection() = (isBuy=true, ok=true)
→ 用 USDT 买 SHIB

4. 转为数据库模型：
WalletTrade {
  ChainID: 1,
  TxHash: "0xabc...",
  WalletAddress: "0xwallet456...",
  DEXName: "uniswap",
  DEXVersion: "v3",
  TokenIn: "0xusdt...",
  TokenOut: "0xshib...",
  AmountUSD: 1000.0,
  IsBuy: true,
  PnlUSD: NULL,  // 稍后计算
}

5. 插入数据库：
INSERT INTO wallet_trades (...) VALUES (...)
ON DUPLICATE KEY UPDATE ...
```

---

## 六、性能与优化

### 1. 并发处理
- V2 和 V3 **顺序执行**（避免 API 限流）
- 历史同步与增量同步**并行**（goroutine）

### 2. 批量大小
- 默认 `batchSize = 1000`
- 可根据 API 响应速度调整

### 3. 错误处理
- 单个 Swap 解析失败 → 跳过，继续
- V2 同步失败 → 记录错误，继续 V3
- 整体失败 → 记录到 sync_log，下次重试

### 4. 避免重复
- 唯一键：`tx_hash + wallet_address + token_out`
- Upsert 自动去重

---

## 七、监控与调试

### 查看同步进度
```sql
-- 最近 5 次同步
SELECT sync_type, start_time, end_time, 
       records_inserted, records_updated, records_skipped,
       TIMESTAMPDIFF(SECOND, start_time, completed_at) as duration_sec
FROM sync_log 
ORDER BY completed_at DESC 
LIMIT 5;
```

### 查看数据分布
```sql
-- V2 vs V3
SELECT dex_version, is_buy, COUNT(*) 
FROM wallet_trades 
GROUP BY dex_version, is_buy;

-- 时间分布
SELECT DATE(block_time) as date, COUNT(*) 
FROM wallet_trades 
GROUP BY date 
ORDER BY date DESC 
LIMIT 7;
```

### 日志关键字
```bash
# 监控同步
tail -f worker.log | grep "Sync completed"

# 错误排查
tail -f worker.log | grep "ERROR"
```

---

## 八、常见问题

### Q1: 为什么 V2 拉不到数据？
**A:** 检查：
1. `skip >= 5000` 触发限制 → 缩短时间范围
2. `minAmountUSD` 过高 → 降低阈值
3. 时间戳是否正确（UTC）

### Q2: 为什么有些卖出没有 PNL？
**A:** 因为对应的买入不在数据库中
- 买入发生在同步范围之外
- 解决：增大 `retention_days` 或初次全量同步更长历史

### Q3: 稳定币互换为什么被过滤？
**A:** USDT ↔ USDC 对聪明钱分析无意义
- 不涉及选币决策
- 可能是套利机器人

### Q4: 如何加快历史数据同步？
**A:** 
1. 提高 `batchSize`（如 2000）
2. 使用更快的 RPC 节点
3. 部署到与 The Graph Gateway 相同区域

---

## 九、总结

### 核心流程
```
1. Worker 定时启动
2. 调用 V2/V3 Client 查询 The Graph
3. GraphQL 分页拉取 Swaps
4. 解析 + 过滤稳定币 + 判断方向
5. 转换为统一模型
6. Upsert 到 wallet_trades
7. 记录 sync_log
```

### 关键设计
- **双客户端**：V2 和 V3 独立处理
- **游标分页**：V3 无限制，V2 有 skip 限制
- **智能过滤**：稳定币互换、方向判断、地址规范化
- **幂等性**：Upsert 支持重复同步
- **可观测**：详细日志 + sync_log 表

### 数据质量保证
- ✅ 去重（唯一键）
- ✅ 过滤无效交易（稳定币互换）
- ✅ 保留已计算 PNL（Omit）
- ✅ 地址小写统一
- ✅ 错误隔离（单条失败不影响整体）

这就是完整的 The Graph 数据拉取逻辑！
