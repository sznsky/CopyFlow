# CopyFlow - 聪明钱包分析平台

链上智能交易分析平台：从已知高质量钱包出发，自动评分并产出代币买入信号。

技术栈：**Go 1.22** + **Gin** + **MySQL 8** + **React 18** + **TypeScript** + **wagmi**

---

## 核心功能

### 聪明钱分析（Smart Money Analytics）
- **种子地址模式**：手动维护已知大户地址（Nansen / Arkham 公开榜单），精准拉取数据
- **The Graph 数据源**：Uniswap V2/V3 子图，按钱包地址过滤，查询量极小
- **6 维评分**：累计盈亏、胜率、盈亏比、最大回撤、主流币占比、交易频率，综合 0-100 分
- **Top 20 榜单**：综合分 ≥ 60 进入榜单，动态排名
- **代币信号**：统计高分钱包近 3 天的共识买入行为，产出买入信号

### 跟单系统（Copy Trading）
- **实时监听**：区块扫描监听领头地址的 DEX 买入交易（BSC / PancakeSwap）
- **策略执行**：固定金额或等比例跟单，自定义滑点和单笔限额
- **链上广播**：AES-256 加密存储私钥，自动签名并广播交易

### Web3 集成
- MetaMask 钱包连接 + SIWE 签名登录
- JWT 鉴权，支持移动端响应式布局

---

## 快速开始

### 前置条件
- MySQL 8.0+，Go 1.22+，Node.js 18+
- [The Graph API Key](https://thegraph.com/studio/)（免费注册，10 万次查询/月）

### 1. 数据库初始化

```bash
mysql -u root -p -e "CREATE DATABASE copyflow DEFAULT CHARSET utf8mb4;"
mysql -u root -p copyflow < backend/migrations/001_init.sql
mysql -u root -p copyflow < backend/migrations/002_auth_email.sql
mysql -u root -p copyflow < backend/migrations/003_smart_money.sql
```

### 2. 配置文件

编辑 `backend/config/config.yaml`，填入必要配置：

```yaml
database:
  dsn: "root:your_password@tcp(127.0.0.1:3306)/copyflow?charset=utf8mb4&parseTime=True&loc=Local"

thegraph:
  enabled: true
  api_key: "your-api-key-here"   # 从 https://thegraph.com/studio/ 获取
  uniswap_v2_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
  uniswap_v3_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV"

smartmoney:
  enabled: true
  chain_id: 1
  min_amount_usd: 500
  eval_days: 30
  seed_wallets:                  # 手动维护的种子钱包地址
    - "0x..."
    - "0x..."
```

### 3. 启动服务

```bash
# 终端 1 - API 服务
cd backend && go run ./cmd/api

# 终端 2 - SmartMoney Worker（定时同步 + 评分 + 信号）
cd backend && go run ./cmd/smartmoney-worker

# 终端 3 - 前端开发服务
cd frontend && pnpm install && pnpm run dev
```

浏览器访问：`http://localhost:5173`

---

## 项目结构

```
CopyFlow/
├── backend/
│   ├── cmd/
│   │   ├── api/                     # HTTP API 服务入口
│   │   ├── worker/                  # 链上跟单监听 Worker
│   │   └── smartmoney-worker/       # 聪明钱分析 Worker
│   ├── internal/
│   │   ├── smartmoney/              # 同步 / 评分 / 信号聚合
│   │   ├── handler/                 # HTTP 处理器
│   │   ├── model/                   # GORM 数据模型
│   │   ├── listener/                # 链上事件监听
│   │   ├── executor/                # 跟单执行
│   │   └── config/                  # 配置结构
│   ├── pkg/
│   │   └── thegraph/                # The Graph GraphQL 客户端
│   ├── migrations/                  # 数据库 SQL 脚本
│   └── config/config.yaml           # 主配置文件
└── frontend/
    └── src/
        ├── pages/
        │   ├── Dashboard.tsx        # 首页信号展示
        │   ├── SmartWallets.tsx     # 聪明钱包榜单
        │   ├── TokenSignals.tsx     # 代币买入信号
        │   ├── WalletDetail.tsx     # 钱包详情
        │   ├── Configs.tsx          # 跟单策略配置
        │   └── Trades.tsx           # 跟单记录
        └── components/
            ├── Layout.tsx           # 导航 + 钱包连接
            └── ConnectWallet.tsx    # MetaMask + SIWE 登录
```

---

## 文档索引

| 文档 | 内容 |
|------|------|
| [SMARTMONEY.md](SMARTMONEY.md) | 聪明钱系统详细说明（评分算法、同步模式、API） |
| [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) | 完整实现总结和启动流程 |
| [PROJECT_STATUS.md](PROJECT_STATUS.md) | 当前完成状态和后续规划 |
| [backend/docs/THEGRAPH_INTEGRATION.md](backend/docs/THEGRAPH_INTEGRATION.md) | The Graph 集成说明 |
