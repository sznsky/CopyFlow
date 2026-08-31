# CopyFlow 聪明钱包系统 - 实现说明

## 项目简介

CopyFlow 是一个链上聪明钱包分析平台。核心功能是**识别、评分和展示高胜率的链上交易钱包**，帮助用户发现值得参考的交易信号。

技术栈：Go 1.22 + Gin + MySQL 8 + React 18 + TypeScript + wagmi

---

## 聪明钱包判断逻辑

### 数据来源：种子地址 + The Graph

系统采用**种子钱包模式**：手动维护一批已知的高质量钱包地址（来源：Nansen Smart Money、Arkham Intelligence 等公开榜单），通过 The Graph 的 Uniswap V2/V3 子图拉取这些地址的历史交易记录。

```
config.yaml seed_wallets（手动维护）
        ↓
The Graph Uniswap V2/V3（按钱包地址过滤）
        ↓
wallet_trades 表（原始 Swap 记录）
        ↓
CalculatePNLForTrades（FIFO 盈亏计算）
        ↓
CalculateWalletScores（6 维评分 0-100）
        ↓
smart_wallets 表（排名 + Top 20 标记）
        ↓
AggregateTokenSignals（共识买入信号）
```

### 6 维评分算法

| 维度 | 满分 | 分档标准 |
|------|------|---------|
| 累计盈亏 PNL | 30 | >$10k=30，>$5k=20，>$1k=10，>$0=5 |
| 胜率 | 20 | ≥70%=20，≥60%=15，≥50%=10，≥40%=5 |
| 盈亏比 | 15 | ≥3=15，≥2=10，≥1.5=5 |
| 最大回撤 | 15 | <10%=15，<20%=10，<30%=5（越小越好） |
| 主流币占比 | 10 | ≥50%=10，≥30%=5 |
| 交易频率/天 | 10 | >5=10，>2=5，>0.5=2 |

评估窗口：最近 **30 天**（可配置）。综合分 ≥60 进入聪明钱包榜单，Top 20 标记 `is_top_wallet=true`。

### 信号聚合

统计近 3 天内，多个高分钱包（score≥60）共同买入了哪些代币，产出**共识买入信号**。共识分由四个因素加权：买入钱包占比（40%）+ 买入总量（30%）+ 高分钱包质量（20%）+ 时间集中度（10%）。

---

## 数据同步模式

Worker 支持两种模式，通过配置自动切换：

| 模式 | 触发条件 | 说明 |
|------|---------|------|
| **种子模式**（推荐） | `seed_wallets` 非空 | 按指定钱包地址过滤，精准拉取，查询量极小 |
| 交易对模式 | `seed_wallets` 为空，`pairs` 非空 | 按 pair 地址过滤（ETH/USDT、ETH/USDC） |
| 全量模式 | 两者均为空 | 拉取所有大额 Swap，不推荐（数据量大） |

---

## 配置说明

编辑 `backend/config/config.yaml`：

```yaml
# The Graph 子图（需要 API Key）
thegraph:
  enabled: true
  api_key: "your-api-key-here"   # 从 https://thegraph.com/studio/ 免费获取
  uniswap_v2_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
  uniswap_v3_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV"

# 聪明钱配置
smartmoney:
  enabled: true
  chain_id: 1            # Ethereum
  min_amount_usd: 500    # 只抓 $500 以上的 Swap
  eval_days: 30          # 评估窗口天数
  min_wallet_score: 60   # 进入榜单的最低评分
  top_wallet_count: 20   # Top N 钱包数量
  signal_days: 3         # 信号聚合分析最近 N 天
  sync_interval_hours: 24

  # 种子钱包地址（来源：Nansen、Arkham 等公开榜单，手动维护）
  seed_wallets:
    - "0xd8da6bf26964af9d7eed9e03e53415d37aa96045"  # 示例地址
    - "0x..."  # 在此添加更多已知聪明钱包
```

---

## 启动流程

```bash
# 1. 数据库初始化
mysql -u root -p copyflow < backend/migrations/001_init.sql
mysql -u root -p copyflow < backend/migrations/002_copy_trading.sql
mysql -u root -p copyflow < backend/migrations/003_smart_money.sql

# 2. 填入 The Graph API Key（config.yaml）

# 3. 填入种子钱包地址（config.yaml seed_wallets）

# 4. 启动 API 服务
cd backend && go run ./cmd/api

# 5. 启动 SmartMoney Worker（定时同步 + 评分）
cd backend && go run ./cmd/smartmoney-worker

# 6. 启动前端
cd frontend && pnpm run dev
```

启动 Worker 后，日志中确认模式：
```
mode=seed_wallets  seed_wallets_count=N   # 种子模式
mode=pair_filter                          # 交易对模式
```

---

## 核心文件结构

```
backend/
├── cmd/
│   ├── api/main.go                        # HTTP API 入口
│   └── smartmoney-worker/main.go          # 定时同步 + 评分 Worker
├── internal/
│   ├── smartmoney/
│   │   ├── sync.go                        # 数据同步（支持种子/pair/全量三种模式）
│   │   ├── scoring.go                     # 6 维评分算法 + FIFO 盈亏计算
│   │   └── signals.go                     # 代币共识信号聚合
│   ├── handler/
│   │   └── smartmoney.go                  # REST API 处理器
│   ├── model/models.go                    # SmartWallet / WalletTrade / TokenSignal
│   └── config/config.go                   # 配置结构（含 SeedWallets / EvalDays）
├── pkg/
│   └── thegraph/
│       ├── client.go                      # The Graph GraphQL 客户端
│       ├── uniswap.go                     # V2/V3 查询（支持 pair_in / origin_in / to_in）
│       └── stablecoins.go                 # 稳定币识别 + 买卖方向判断
└── config/config.yaml                     # 主配置文件

frontend/
└── src/
    ├── pages/
    │   ├── Dashboard.tsx                  # 首页（信号 + 聪明钱包动态）
    │   ├── SmartWallets.tsx               # 聪明钱包榜单
    │   ├── TokenSignals.tsx               # 代币买入信号
    │   └── WalletDetail.tsx               # 单个钱包详情 + 交易记录
    ├── components/
    │   ├── Layout.tsx                     # 导航（含钱包连接）
    │   └── ConnectWallet.tsx              # MetaMask 连接 + SIWE 签名登录
    └── context/AuthContext.tsx            # 全局认证状态
```

---

## REST API

```
GET  /api/smart-wallets              # 获取高分钱包列表（支持 limit / min_score 参数）
GET  /api/token-signals              # 获取代币共识信号（支持 min_score / days 参数）
GET  /api/token-signals/:id/details  # 获取某个信号的详细买入记录
GET  /api/wallet-history/:address    # 获取某个钱包的交易历史

POST /api/admin/sync                 # 手动触发数据同步（需要 JWT）
POST /api/admin/calculate-scores     # 手动触发评分计算（需要 JWT）
POST /api/admin/aggregate-signals    # 手动触发信号聚合（需要 JWT）
```

---

## 常见问题

**Q: 种子钱包地址从哪里找？**
A: 推荐来源：
- [Nansen Smart Money](https://app.nansen.ai/smart-money)（公开榜单）
- [Arkham Intelligence](https://intel.arkm.com/)（已识别标签地址）
- Etherscan 上标记为 DEX 大户的地址
- GitHub 上整理的 DeFi 大户地址列表

**Q: The Graph API Key 怎么获取？**
A: 访问 https://thegraph.com/studio/ 免费注册，免费层每月 10 万次查询。种子模式下每天只需几十次查询，完全够用。

**Q: 评估窗口为什么是 30 天而不是更长？**
A: 30 天可以快速验证系统，拉取数据量小，评分反映近期表现。生产环境可调整 `eval_days` 到 90 天。

**Q: 如何修改评分算法？**
A: 编辑 `backend/internal/smartmoney/scoring.go` 中的 `calculateCompositeScore` 函数。

**Q: 为什么没有跟单功能？**
A: 当前版本专注于**信号展示**，帮助用户发现值得参考的交易机会，用户自行决策。跟单执行涉及链上资产风险，作为独立模块在后续版本规划中。
