# 聪明钱分析功能

本项目已经从跟单系统升级为聪明钱分析平台，帮助用户发现高绩效交易者并跟踪他们的投资信号。

## 功能概览

### 1. 数据源集成
- 从 **The Graph** 子图获取 Uniswap V2/V3 交易数据
- 通过 GraphQL 直接查询 `swaps` 事件，无需自建查询/等待索引
- 筛选条件：交易金额 ≥ 配置的 `min_amount_usd`（默认 10000 USD）
- 自动过滤稳定币互换（USDT/USDC/DAI 等）与黄金代币互换，详见 `backend/pkg/thegraph/stablecoins.go`
- 注意：当前主要支持 Ethereum 主网（Uniswap V2/V3 子图）

### 2. 钱包评分系统
基于过去 6 个月的历史交易，从以下维度评分（0-100分）：
- **累计盈亏**（30分）：总盈利能力
- **胜率**（20分）：盈利交易占比
- **盈亏比**（15分）：平均盈利 / 平均亏损
- **最大回撤**（15分）：风险控制能力
- **主流币占比**（10分）：交易主流币（ETH、BTC、USDT等）的比例
- **交易频率**（10分）：每日平均交易次数

### 3. Top 20 钱包筛选
- 按综合评分排名
- 最低评分阈值：60分
- 实时更新排名

### 4. 信号聚合
分析高分钱包最近 3 天的买入行为：
- 统计共同买入的代币
- 计算共识度评分（0-100分）
- 包含：买入钱包数、总买入量、平均买入金额、时间分布等

## 快速开始

### 1. 配置 The Graph

1. 访问 https://thegraph.com/studio/ 注册账户并创建 API Key
2. 编辑 `backend/config/config.yaml`，填入 API Key（**同时替换 endpoint URL 中的 `[api-key]`**）：

```yaml
# The Graph 子图配置
thegraph:
  enabled: true
  api_key: "your-api-key-here"
  uniswap_v2_endpoint: "https://gateway.thegraph.com/api/your-api-key-here/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
  uniswap_v3_endpoint: "https://gateway.thegraph.com/api/your-api-key-here/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV"

# 聪明钱配置
smartmoney:
  enabled: true
  chain_id: 1  # Ethereum
  min_amount_usd: 10000
  top_wallet_count: 20
  min_wallet_score: 60
  signal_days: 3
  sync_interval_hours: 24
  batch_size: 1000       # The Graph 分页大小
```

The Graph 的 Uniswap V2/V3 子图 schema 固定，无需自建查询；GraphQL 查询与字段映射详见 [backend/docs/THEGRAPH_INTEGRATION.md](backend/docs/THEGRAPH_INTEGRATION.md)。

### 2. 初始化数据库

```bash
mysql -u root -p copyflow < backend/migrations/003_smart_money.sql
```

### 3. 启动服务

```bash
# 启动 API 服务
cd backend && go run ./cmd/api

# 启动聪明钱 Worker（自动拉取历史数据 + 定期增量同步）
cd backend && go run ./cmd/smartmoney-worker

# 启动前端
cd frontend && npm run dev
```

### 4. 手动触发同步（可选）

```bash
# 同步交易数据（The Graph）
curl -X POST http://localhost:8080/api/admin/sync \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"start_date": "2026-06-01", "end_date": "2026-06-20", "min_amount_usd": 10000}'

# 计算钱包评分
curl -X POST http://localhost:8080/api/admin/calculate-scores \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 聚合代币信号
curl -X POST http://localhost:8080/api/admin/aggregate-signals \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"days": 3}'
```

## API 接口

### 用户接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/smart-wallets` | 获取高分钱包列表 |
| GET | `/api/token-signals` | 获取代币信号列表 |
| GET | `/api/token-signals/:id/details` | 获取信号详情 |
| GET | `/api/wallet-history/:address` | 获取钱包交易历史 |

### 管理员接口（需要登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/sync` | 手动触发 The Graph 数据同步 |
| POST | `/api/admin/calculate-scores` | 手动触发评分计算 |
| POST | `/api/admin/aggregate-signals` | 手动触发信号聚合 |

## 前端页面

1. **聪明钱包** (`/smart-wallets`)
   - 查看 Top 20 高分钱包
   - 查看评分、盈亏、胜率等指标
   - 点击查看钱包详情

2. **代币信号** (`/token-signals`)
   - 查看高分钱包共识买入的代币
   - 查看共识度、买入数量、时间等信息
   - 点击查看信号详情

3. **钱包详情** (`/smart-wallets/:address`)
   - 查看钱包完整交易历史
   - 包含买入/卖出、盈亏、持仓时长等

## 工作流程

```
1. SmartMoney Worker 定期运行（默认24小时）
   ↓
2. 从 The Graph 同步 Uniswap V2/V3 交易数据
   ↓
3. 计算每笔交易的盈亏（匹配买入/卖出）
   ↓
4. 计算所有钱包的综合评分
   ↓
5. 筛选 Top 20 钱包
   ↓
6. 分析最近3天的买入行为
   ↓
7. 聚合代币信号，计算共识度
   ↓
8. 前端展示结果
```

## 评分算法

### 钱包评分（0-100分）

1. **累计盈亏**（30分）
   - > $10,000: 30分
   - > $5,000: 20分
   - > $1,000: 10分
   - > $0: 5分

2. **胜率**（20分）
   - ≥ 70%: 20分
   - ≥ 60%: 15分
   - ≥ 50%: 10分
   - ≥ 40%: 5分

3. **盈亏比**（15分）
   - ≥ 3: 15分
   - ≥ 2: 10分
   - ≥ 1.5: 5分

4. **最大回撤**（15分）
   - < 10%: 15分
   - < 20%: 10分
   - < 30%: 5分

5. **主流币占比**（10分）
   - ≥ 50%: 10分
   - ≥ 30%: 5分

6. **交易频率**（10分）
   - > 5次/天: 10分
   - > 2次/天: 5分
   - > 0.5次/天: 2分

### 共识度评分（0-100分）

1. **钱包数量占比**（40分）
   - 买入钱包数 / Top 20 钱包数

2. **总买入量**（30分）
   - > $100,000: 30分
   - > $50,000: 20分
   - > $10,000: 10分
   - > $5,000: 5分

3. **高分钱包权重**（20分）
   - 平均分 > 90: 20分
   - 平均分 > 80: 15分
   - 平均分 > 70: 10分
   - 平均分 > 60: 5分

4. **时间集中度**（10分）
   - 24小时内: 10分
   - 48小时内: 5分
   - 72小时内: 2分

## 注意事项

1. **The Graph API 限制**
   - Gateway 按查询付费（免费额度有限），注意监控用量（https://thegraph.com/studio/ 仪表盘）
   - V2 子图使用 `skip` 分页，`skip` 上限为 5000，超出需改用按时间窗口分批拉取
   - 建议合理设置 `batch_size` 与同步间隔，避免请求过于频繁

2. **数据质量**
   - 依赖 Uniswap V2/V3 官方子图的索引进度，链上确认与子图索引之间存在延迟
   - 确保交易数据包含买入和卖出记录
   - 盈亏计算依赖于匹配的买卖对（FIFO 匹配）

3. **性能优化**
   - 首次同步（180 天历史数据）可能需要较长时间，Worker 会在后台分批拉取
   - 可以根据需要调整评估周期、信号周期与 `retention_days`

4. **链支持**
   - 当前主要支持 Ethereum（Uniswap V2/V3 子图）
   - 可以扩展到其他 EVM 链，需要在配置中新增对应 DEX 子图的 endpoint

## 扩展方向

- [ ] 增加更多评分维度（如交易时间分布、代币多样性等）
- [ ] 支持自定义评分权重
- [ ] 增加风险提示和预警
- [ ] 集成代币元数据（CoinGecko、DEXTools等）
- [ ] 支持 Telegram/Discord 信号推送
- [ ] 增加回测功能
- [ ] 支持更多 DEX 子图（如 SushiSwap、Curve 等）

## 常见问题

**Q: 如何获取 The Graph API Key?**  
A: 访问 https://thegraph.com/studio/ 注册/登录后创建 API Key，并将其填入 `config.yaml` 的 `thegraph.api_key`，同时替换 endpoint URL 中的 `[api-key]` 占位符

**Q: Uniswap V2/V3 子图的 endpoint 在哪里找？**  
A: 默认使用官方子图 ID（V2: `EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu`，V3: `5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV`），也可以在 https://thegraph.com/explorer 搜索其他子图替换

**Q: 为什么没有显示钱包数据？**  
A: 确保已启用 `thegraph` 和 `smartmoney` 配置，并运行了 Worker 或手动触发了同步（首次同步需要等待 180 天历史数据拉取完成）

**Q: 如何修改评分算法？**  
A: 编辑 `backend/internal/smartmoney/scoring.go` 中的评分逻辑

**Q: 可以只关注特定代币吗？**  
A: 可以在信号聚合或前端展示时按代币地址过滤，也可以调整 `min_amount_usd` 阈值缩小交易范围
