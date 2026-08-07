# Smart Money Worker - The Graph 数据同步

快速启动指南

## 前置要求

1. MySQL 数据库已启动
2. 已执行所有迁移脚本（001~003，聪明钱相关表结构在 003_smart_money.sql 中一次性创建）
3. 获取 The Graph API Key：https://thegraph.com/studio/

## 配置

编辑 `backend/config/config.yaml`：

```yaml
thegraph:
  enabled: true
  api_key: "YOUR_API_KEY_HERE"  # 替换为你的 API Key
  # 同时替换 endpoint URL 中的 [api-key]
  uniswap_v2_endpoint: "https://gateway.thegraph.com/api/YOUR_API_KEY_HERE/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
  uniswap_v3_endpoint: "https://gateway.thegraph.com/api/YOUR_API_KEY_HERE/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV"

smartmoney:
  enabled: true
  min_amount_usd: 10000  # 最小交易金额，可根据需要调整
```

## 启动

```bash
cd backend
go run ./cmd/smartmoney-worker
```

## 工作流程

Worker 会自动：
1. 后台拉取最近 180 天历史数据（分批，每次 30 天）
2. 每天增量同步昨天的交易数据
3. 计算 PNL、评分、聚合信号
4. 清理超过 180 天的旧数据

## 监控日志

```
[SmartMoney Worker] Starting historical sync (6 months)
[SmartMoney Worker] Starting incremental cycle
[SmartMoney] Syncing Uniswap V2...
[SmartMoney] Syncing Uniswap V3...
[SmartMoney] Sync completed: X inserted, Y updated, Z skipped
```

## 数据验证

查看同步结果：
```sql
-- 查看同步日志
SELECT * FROM sync_log ORDER BY created_at DESC LIMIT 10;

-- 查看交易数据
SELECT COUNT(*), dex_version FROM wallet_trades GROUP BY dex_version;

-- 查看评分数据
SELECT COUNT(*) FROM smart_wallets WHERE score > 60;
```

## 手动触发同步（通过 API）

```bash
curl -X POST http://localhost:8080/api/admin/sync \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2026-06-01",
    "end_date": "2026-06-20",
    "min_amount_usd": 10000
  }'
```

详细文档见 `docs/THEGRAPH_INTEGRATION.md`
