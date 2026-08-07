# CopyFlow - 聪明钱分析平台

链上智能交易分析平台：发现高绩效交易者，跟踪投资信号，自动跟单执行。

## 核心功能

### 🔍 聪明钱分析（Smart Money Analytics）
- **数据源**: The Graph 子图集成，获取 Uniswap V2/V3 等 DEX 的链上交易数据
- **钱包评分**: 基于 6 个月历史交易，多维度评分（盈亏、胜率、盈亏比、回撤等）
- **Top 20 筛选**: 自动筛选高分钱包，实时排名
- **信号聚合**: 分析高分钱包的共识买入行为，生成投资信号

### 🤖 自动跟单（Copy Trading）
- **实时监听**: 监听指定领头地址的 DEX 买入交易
- **策略执行**: 支持固定金额/等比例跟单，自定义滑点、限额
- **多链支持**: BSC、Ethereum（可扩展更多 EVM 链）
- **安全私钥**: AES-256 加密存储

## 快速开始

### 前置条件
- MySQL 8.0+
- Go 1.22+
- Node.js 18+
- The Graph API Key（聪明钱功能，从 https://thegraph.com/studio/ 获取）

### 1. 数据库初始化

```bash
mysql -u root -p -e "CREATE DATABASE copyflow DEFAULT CHARSET utf8mb4;"
mysql -u root -p copyflow < backend/migrations/001_init.sql
mysql -u root -p copyflow < backend/migrations/002_auth_email.sql
mysql -u root -p copyflow < backend/migrations/003_smart_money.sql
```

### 2. 配置文件

编辑 `backend/config/config.yaml`：

```yaml
database:
  dsn: "root:你的密码@tcp(127.0.0.1:3306)/copyflow?charset=utf8mb4&parseTime=True&loc=Local"

# The Graph 子图配置（聪明钱分析）
thegraph:
  enabled: true
  api_key: "your-api-key-here"  # 从 https://thegraph.com/studio/ 获取
  uniswap_v2_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu"
  uniswap_v3_endpoint: "https://gateway.thegraph.com/api/[api-key]/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV"

# 聪明钱配置
smartmoney:
  enabled: true
  chain_id: 1  # Ethereum
  min_amount_usd: 1000
  top_wallet_count: 20
  signal_days: 3
```

详细配置说明：见 [SMARTMONEY.md](SMARTMONEY.md)

### 3. 启动服务

```bash
# 终端 1 - API 服务
cd backend && go run ./cmd/api

# 终端 2 - 跟单 Worker（可选）
cd backend && go run ./cmd/worker

# 终端 3 - 聪明钱 Worker
cd backend && go run ./cmd/smartmoney-worker

# 终端 4 - 前端
cd frontend && npm install && npm run dev
```

浏览器访问: `http://localhost:5173`

## 项目结构

```
CopyFlow/
├── backend/
│   ├── cmd/
│   │   ├── api/                 # HTTP API 服务
│   │   ├── worker/              # 跟单监听 Worker
│   │   └── smartmoney-worker/  # 聪明钱分析 Worker
│   ├── internal/
│   │   ├── smartmoney/          # 聪明钱分析核心逻辑
│   │   ├── handler/             # HTTP 请求处理
│   │   ├── model/               # 数据模型
│   │   └── ...
│   ├── pkg/
│   │   ├── thegraph/             # The Graph GraphQL 客户端
│   │   └── ...
│   ├── migrations/              # 数据库迁移
│   └── config/                  # 配置文件
├── frontend/                    # React 前端
│   ├── src/
│   │   ├── pages/
│   │   │   ├── SmartWallets.tsx      # 聪明钱包列表
│   │   │   ├── TokenSignals.tsx      # 代币信号
│   │   │   ├── WalletDetail.tsx      # 钱包详情
│   │   │   └── ...
│   │   └── ...
└── docs/                        # 文档
```

## 核心特性

### 聪明钱分析
- 🎯 **多维度评分**: 累计盈亏、胜率、盈亏比、最大回撤、主流币占比、交易频率
- 📊 **实时排名**: Top 20 高分钱包动态更新
- 🔥 **信号聚合**: 分析共识买入行为，计算共识度评分
- 📈 **完整历史**: 查看钱包完整交易记录，包含盈亏分析

### 自动跟单
- ⚡ **实时监听**: 区块链实时监听领头地址交易
- 🎛️ **灵活策略**: 固定金额/等比例跟单，自定义滑点和限额
- 🔐 **安全可靠**: 私钥 AES-256 加密存储
- 🌐 **多链支持**: BSC、Ethereum（可扩展）

### 用户体验
- 🔑 **多种登录**: 支持钱包签名登录、邮箱登录
- 📱 **响应式设计**: 适配桌面和移动端
- 🎨 **现代 UI**: 基于 React + Tailwind CSS

## 文档索引

| 文档 | 内容 |
|------|------|
| [SMARTMONEY.md](SMARTMONEY.md) | **聪明钱功能详细说明** |
| [backend/README.md](backend/README.md) | 后端架构、配置、API 接口 |
| [frontend/README.md](frontend/README.md) | 前端页面、目录结构 |
| [backend/docs/THEGRAPH_INTEGRATION.md](backend/docs/THEGRAPH_INTEGRATION.md) | The Graph 集成说明 |
