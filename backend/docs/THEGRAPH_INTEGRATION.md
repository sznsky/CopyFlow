# The Graph 集成说明

## 概述

已成功将数据源从 Dune Analytics 迁移到 The Graph，用于 Smart Money 分析功能。

## 主要变更

### 1. 删除的文件/代码
- ✅ `pkg/dune/client.go` - Dune 客户端
- ✅ 所有 `SyncTradesFromDune` 相关代码
- ✅ `DuneSyncLog` 模型（替换为 `SyncLog`）
- ✅ 配置中的 `dune.*` 配置项

### 2. 新增的文件/功能
- ✅ `pkg/thegraph/client.go` - The Graph GraphQL 客户端
- ✅ `pkg/thegraph/stablecoins.go` - 稳定币识别与交易方向判断
- ✅ `pkg/thegraph/uniswap.go` - Uniswap V2/V3 查询与解析
- ✅ `internal/smartmoney/sync.go` - 重构的同步逻辑
- ✅ `migrations/003_smart_money.sql` - 已合并为直接创建 The Graph 版最终表结构（`sync_log` 表、`wallet_trades.dex_version` 列等），不再包含 Dune 相关表

### 3. 配置变更

#### config.yaml 新增配置：

```yaml
thegraph:
  enabled: true
  api_key: "your-api-key-here"  # 需要从 The Graph Studio 获取
  uniswap_v2_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
  uniswap_v3_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV"

smartmoney:
  enabled: true
  chain_id: 1  # Ethereum
  min_amount_usd: 10000  # 最小交易金额（USD）
  retention_days: 180    # 数据保留天数（半年）
  batch_size: 1000       # The Graph 分页大小
```

## 获取 The Graph API Key

1. 访问 https://thegraph.com/studio/
2. 登录或注册账户
3. 创建 API Key
4. 将 API Key 替换到 `config.yaml` 中的 `api_key` 字段
5. **重要**：将 endpoint URL 中的 `[api-key]` 也替换为你的 API Key

示例：
```yaml
api_key: "abc123def456"
uniswap_v2_endpoint: "https://gateway.thegraph.com/api/abc123def456/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
```

## 数据同步机制

### 两个并行任务：

1. **历史数据同步（只执行一次）**
   - 拉取最近 180 天的历史数据
   - 分批拉取，每次 30 天
   - 后台运行，不阻塞主流程

2. **增量同步（定时执行）**
   - 每天拉取昨天的新数据
   - 默认 24 小时运行一次
   - 执行完整的分析流程：同步 → PNL 计算 → 评分 → 信号聚合
   - 清理超过 180 天的旧数据

### 数据流程：

```
The Graph (Uniswap V2/V3)
    ↓
过滤稳定币互换
    ↓
判断买入/卖出方向
    ↓
存入 wallet_trades
    ↓
FIFO 匹配计算 PNL
    ↓
计算钱包评分 (6个月数据)
    ↓
聚合代币信号 (Top 20 钱包最近3天买入)
```

## 启动服务

### 1. 更新配置
编辑 `backend/config/config.yaml`，填入 The Graph API Key。

### 2. 启动 Worker
```bash
cd backend
go run ./cmd/smartmoney-worker
```

### 3. 启动 API（可选）
```bash
go run ./cmd/api
```

## 查询效率

- **V2**: 使用 `skip` 分页，限制 5000 条
- **V3**: 使用 ID 游标分页，无限制
- **批量大小**: 默认 1000 条/批次
- **金额过滤**: 在 GraphQL 查询中直接过滤 `amountUSD >= min_amount_usd`

## 稳定币过滤

系统会自动过滤以下类型的交易：
- USDT ↔ USDC
- USDT ↔ DAI
- 其他稳定币之间的互换
- 黄金代币（PAXG、XAUT）互换

完整列表见 `pkg/thegraph/stablecoins.go`。

## 字段映射

| The Graph | WalletTrade 模型 | 说明 |
|-----------|-----------------|------|
| transaction.id | tx_hash | 交易哈希 |
| transaction.timestamp | block_time | 区块时间 |
| transaction.blockNumber | block_number | 区块高度 |
| pair.id / pool.id | pool_address | 池子地址 |
| to / origin | wallet_address | 钱包地址（V2 用 to，V3 用 origin） |
| token0, token1 | token_in, token_out | 根据金额正负判断 |
| amountUSD | amount_usd | 交易金额（USD） |
| - | dex_name | 固定为 "uniswap" |
| - | dex_version | "v2" 或 "v3" |

## 数据库迁移

`migrations/003_smart_money.sql` 已直接创建 The Graph 版最终表结构：
- ✅ `wallet_trades` 表包含 `dex_version` 列
- ✅ `sync_log` 表（记录 The Graph 同步日志）
- ✅ 不再包含任何 Dune 相关表（原 `dune_sync_log` 已移除）

新环境只需依次执行 `001_init.sql`、`002_auth_email.sql`、`003_smart_money.sql` 即可。

## 后续建议

1. **监控 The Graph API 用量**：查看 https://thegraph.com/studio/ 仪表盘
2. **调整同步频率**：根据需要修改 `sync_interval_hours`
3. **优化数据保留期**：根据存储和性能需求调整 `retention_days`
4. **添加告警**：监控同步失败、API 限额等异常

## 故障排查

### 常见问题：

1. **API Key 错误**
   - 检查 `config.yaml` 中的 `api_key` 是否正确
   - 确认 endpoint URL 中的 `[api-key]` 已替换

2. **查询超时**
   - 减小 `batch_size`
   - 缩短每次查询的时间范围

3. **数据量过大**
   - 提高 `min_amount_usd` 阈值
   - 减少 `retention_days`

4. **PNL 为空**
   - 等待历史数据同步完成
   - 确认有匹配的买入/卖出交易对

## 维护建议

- 定期检查 `sync_log` 表的同步状态
- 监控 `wallet_trades` 表大小
- 定期备份评分和信号数据
