# CopyFlow 聪明钱包系统

## 概述

聪明钱包系统通过**种子地址 + The Graph** 的方式，自动分析已知高质量钱包的链上交易行为，产出评分和代币买入信号。

---

## 工作原理

### 第一步：维护种子地址

在 `config/config.yaml` 中手动维护一批已知的高质量链上地址：

```yaml
smartmoney:
  seed_wallets:
    - "0xd8da6bf26964af9d7eed9e03e53415d37aa96045"
    - "0x..."  # 更多地址
```

**地址来源建议**：
- [Nansen Smart Money](https://app.nansen.ai/smart-money)
- [Arkham Intelligence](https://intel.arkm.com/)
- Etherscan 上标记的 DEX 大户
- GitHub 整理的 DeFi 交易员地址列表

### 第二步：拉取交易数据

SmartMoney Worker 定时通过 The Graph Uniswap V2/V3 子图，按钱包地址过滤拉取 Swap 记录：

- V2 使用 `to_in` 过滤（`to` 字段为接收方钱包）
- V3 使用 `origin_in` 过滤（`origin` 字段为发起方钱包）
- 只拉取金额 ≥ `min_amount_usd`（默认 $500）的交易
- 评估窗口：最近 `eval_days` 天（默认 30 天）

### 第三步：计算盈亏

对每个钱包的卖出交易，用 **FIFO 原则**匹配对应的买入交易，计算：
- `pnl_usd`：绝对盈亏（USD）
- `pnl_percent`：盈亏百分比
- `holding_duration_hours`：持仓时长

### 第四步：6 维评分

```
综合分（0-100）= PNL 分 + 胜率分 + 盈亏比分 + 回撤分 + 主流币分 + 频率分
```

| 维度 | 满分 | 分档 |
|------|------|------|
| 累计盈亏 | 30 | >$10k=30，>$5k=20，>$1k=10，>$0=5 |
| 胜率 | 20 | ≥70%=20，≥60%=15，≥50%=10，≥40%=5 |
| 盈亏比 | 15 | ≥3=15，≥2=10，≥1.5=5 |
| 最大回撤 | 15 | <10%=15，<20%=10，<30%=5 |
| 主流币占比 | 10 | ≥50%=10，≥30%=5 |
| 交易频率/天 | 10 | >5=10，>2=5，>0.5=2 |

综合分 ≥ `min_wallet_score`（默认 60）进入榜单。

### 第五步：排名和 Top 标记

按综合分 + 累计 PNL 排序，前 `top_wallet_count`（默认 20）名标记 `is_top_wallet=true`。

### 第六步：代币信号聚合

分析近 `signal_days`（默认 3）天内，score ≥ 60 的钱包共同买入的代币，计算**共识分**：

```
共识分 = 买入钱包占比×40 + 买入总量分×30 + 高分钱包质量×20 + 时间集中度×10
```

共识分高的代币作为"今日机会"展示在 Dashboard。

---

## 同步模式

Worker 根据配置自动选择模式：

| 优先级 | 模式 | 触发条件 | 说明 |
|--------|------|---------|------|
| 1 | **种子模式** | `seed_wallets` 非空 | 只拉指定地址，精准高效 |
| 2 | 交易对模式 | `pairs` 非空 | 按 pair 地址过滤 |
| 3 | 全量模式 | 两者均为空 | 拉取所有大额 Swap |

**推荐使用种子模式**：每天只需几十次 The Graph 查询，远低于免费层限额（10 万次/月）。

---

## 配置参数

```yaml
smartmoney:
  enabled: true
  chain_id: 1                  # Ethereum 主网
  min_amount_usd: 500          # 最小交易金额过滤
  eval_days: 30                # 评分评估窗口（天）
  retention_days: 180          # wallet_trades 数据保留天数
  top_wallet_count: 20         # Top N 钱包
  min_wallet_score: 60         # 进入榜单最低分
  signal_days: 3               # 信号聚合窗口（天）
  sync_interval_hours: 24      # 自动同步间隔
  batch_size: 1000             # The Graph 分页大小

  seed_wallets:                # 种子钱包地址列表（手动维护）
    - "0x..."
```

---

## 相关代码

| 功能 | 文件 |
|------|------|
| 数据同步（三种模式） | `internal/smartmoney/sync.go` |
| 6 维评分 + FIFO 盈亏 | `internal/smartmoney/scoring.go` |
| 代币信号聚合 | `internal/smartmoney/signals.go` |
| The Graph 客户端 | `pkg/thegraph/client.go` |
| Uniswap 查询（含钱包/pair 过滤） | `pkg/thegraph/uniswap.go` |
| 稳定币识别 + 买卖方向 | `pkg/thegraph/stablecoins.go` |
| REST API 处理器 | `internal/handler/smartmoney.go` |
| Worker 入口 | `cmd/smartmoney-worker/main.go` |
| 配置结构 | `internal/config/config.go` |
| 数据模型 | `internal/model/models.go` |
| 数据库建表脚本 | `migrations/003_smart_money.sql` |

---

## API 接口

```
GET  /api/smart-wallets
     ?limit=20&min_score=60
     → 返回高分钱包列表（评分、PNL、胜率等）

GET  /api/token-signals
     ?min_score=50&days=3
     → 返回代币共识买入信号

GET  /api/token-signals/:id/details
     → 返回某信号的详细买入记录（哪些钱包买了）

GET  /api/wallet-history/:address
     → 返回某钱包的完整交易历史

POST /api/admin/sync               # 手动触发同步（需 JWT）
POST /api/admin/calculate-scores   # 手动触发评分（需 JWT）
POST /api/admin/aggregate-signals  # 手动触发信号聚合（需 JWT）
```
