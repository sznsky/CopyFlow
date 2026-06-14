# 聪明钱分析功能

本项目已经从跟单系统升级为聪明钱分析平台，帮助用户发现高绩效交易者并跟踪他们的投资信号。

## 功能概览

### 1. 数据源集成
- 从 **Dune Analytics** 获取 Uniswap 交易数据
- 支持 USDC/USDT、ETH/USDT 等主要交易对
- 筛选条件：交易金额 > 1000 USD
- 注意：不同链上的资产（USDT-ETH, USDT-BSC, USDT-ARB）被视为不同资产

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

### 1. 配置 Dune API

编辑 `backend/config/config.yaml`：

```yaml
# Dune Analytics API
dune:
  api_key: "your-dune-api-key-here"
  query_id: 1234567  # 你的 Dune 查询 ID
  enabled: true

# 聪明钱配置
smartmoney:
  enabled: true
  chain_id: 1  # Ethereum
  min_amount_usd: 1000
  top_wallet_count: 20
  min_wallet_score: 60
  signal_days: 3
  sync_interval_hours: 24
```

### 2. 创建 Dune 查询

在 Dune Analytics 创建查询，获取交易数据。查询应该返回以下字段：

- `wallet_address`: 钱包地址
- `tx_hash`: 交易哈希
- `block_number`: 区块号
- `block_time`: 交易时间（ISO 8601格式）
- `dex_name`: DEX 名称（如 "uniswap_v2"）
- `pool_address`: 池子地址
- `token_in`: 输入代币地址
- `token_out`: 输出代币地址
- `token_in_symbol`: 输入代币符号
- `token_out_symbol`: 输出代币符号
- `amount_in`: 输入数量（字符串）
- `amount_out`: 输出数量（字符串）
- `amount_usd`: 交易金额（美元）
- `is_buy`: 是否为买入（布尔值）

参考查询 SQL：见 `backend/docs/dune_query_example.sql`

### 3. 初始化数据库

```bash
mysql -u root -p copyflow < backend/migrations/003_smart_money.sql
```

### 4. 启动服务

```bash
# 启动 API 服务
cd backend && go run ./cmd/api

# 启动聪明钱 Worker（定期同步数据）
cd backend && go run ./cmd/smartmoney-worker

# 启动前端
cd frontend && npm run dev
```

### 5. 手动触发同步（可选）

```bash
# 同步交易数据
curl -X POST http://localhost:8080/api/admin/sync \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query_id": 1234567, "min_amount_usd": 1000}'

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
| POST | `/api/admin/sync` | 手动触发 Dune 数据同步 |
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
2. 从 Dune Analytics 同步交易数据
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

1. **Dune API 限制**
   - 免费账户有请求速率限制
   - 查询执行可能需要几分钟
   - 建议设置合理的同步间隔

2. **数据质量**
   - Dune 查询需要准确返回所需字段
   - 确保交易数据包含买入和卖出记录
   - 盈亏计算依赖于匹配的买卖对

3. **性能优化**
   - 首次同步可能需要较长时间
   - 建议在低峰期运行大量数据同步
   - 可以根据需要调整评估周期和信号周期

4. **链支持**
   - 当前主要支持 Ethereum
   - 可以扩展到其他 EVM 链
   - 需要为不同链创建对应的 Dune 查询

## 扩展方向

- [ ] 增加更多评分维度（如交易时间分布、代币多样性等）
- [ ] 支持自定义评分权重
- [ ] 增加风险提示和预警
- [ ] 集成代币元数据（CoinGecko、DEXTools等）
- [ ] 支持 Telegram/Discord 信号推送
- [ ] 增加回测功能
- [ ] 支持更多数据源（The Graph、Flipside Crypto等）

## 常见问题

**Q: 如何获取 Dune API Key?**  
A: 访问 https://dune.com/settings/api 创建 API Key

**Q: 查询 ID 在哪里找？**  
A: Dune 查询 URL 中的数字，如 `https://dune.com/queries/1234567`

**Q: 为什么没有显示钱包数据？**  
A: 确保已启用 Dune 和 SmartMoney，并运行了 Worker 或手动触发了同步

**Q: 如何修改评分算法？**  
A: 编辑 `backend/internal/smartmoney/scoring.go` 中的评分逻辑

**Q: 可以只关注特定代币吗？**  
A: 可以修改 Dune 查询添加代币过滤条件，或在信号聚合时过滤
