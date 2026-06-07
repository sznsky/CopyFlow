# CopyFlow Backend

Go 后端服务，负责用户鉴权、跟单配置管理，以及链上领头地址监听与自动跟单执行。

## 功能概览

| 进程 | 职责 |
|------|------|
| `cmd/api` | REST API：钱包登录、配置 CRUD、交易查询 |
| `cmd/worker` | 区块扫描 → 解析 Swap → 策略判断 → 广播跟单 → 确认交易 |

**MVP 支持：** BSC + PancakeSwap V2 跟单买入。Ethereum + Uniswap V2 已在配置中预留，默认关闭。

## 架构

```
用户/前端
    │  HTTP
    ▼
┌─────────────┐     ┌──────────────┐
│  API Server │────▶│    MySQL     │
└─────────────┘     └──────────────┘
                           ▲
┌─────────────┐            │
│   Worker    │────────────┘
└──────┬──────┘
       │ RPC
       ▼
┌──────────────────────────────────┐
│  Listener → Strategy → Executor    │
│       ↓                            │
│  PancakeSwap / Uniswap Router      │
└──────────────────────────────────┘
```

**跟单主流程：**

1. Worker 轮询新区块，筛选领头地址发出的 DEX 买入交易
2. 解析 Swap 信息并写入 `leader_trades`
3. 匹配用户的 `copy_configs`，经策略引擎计算跟单金额
4. 解密跟单钱包私钥，构造 Router swap 交易并广播
5. 轮询 receipt，将状态更新为 `success` / `failed`

## 目录结构

```
backend/
├── cmd/
│   ├── api/              # HTTP API 入口
│   └── worker/           # 链上监听 + 跟单执行入口
├── internal/
│   ├── auth/             # SIWE 签名登录 + JWT
│   ├── bootstrap/        # 链 / DEX 注册初始化
│   ├── chain/            # 多链 RPC 抽象（可扩展新链）
│   ├── config/           # 配置加载（YAML + 环境变量）
│   ├── dex/              # DEX 解析 / 执行抽象（可扩展 V3）
│   ├── executor/         # 跟单编排 + 交易确认
│   ├── handler/          # HTTP 请求处理
│   ├── listener/         # 区块扫描、领头地址过滤
│   ├── middleware/       # JWT 鉴权、CORS
│   ├── model/            # 数据模型与常量
│   ├── store/            # MySQL 数据访问
│   └── strategy/         # 跟单策略（比例 / 固定额 / 上限）
├── pkg/crypto/           # 跟单钱包私钥 AES 加密
├── config/config.yaml    # 主配置文件
├── migrations/           # 数据库建表 SQL
└── .env.example          # 环境变量示例
```

## 环境要求

- Go 1.22+
- MySQL 8.0+（本地或远程均可）
- 可访问的 BSC RPC 节点（公开节点或 Alchemy / QuickNode 等）

## 快速开始

### 1. 初始化数据库

```sql
CREATE DATABASE IF NOT EXISTS copyflow DEFAULT CHARSET utf8mb4;
```

```bash
mysql -u root -p copyflow < migrations/001_init.sql
```

### 2. 修改配置

编辑 `config/config.yaml`，至少修改数据库连接：

```yaml
database:
  dsn: "root:你的密码@tcp(127.0.0.1:3306)/copyflow?charset=utf8mb4&parseTime=True&loc=Local"
```

也可用环境变量覆盖，参见 `.env.example`（变量名规则：`database.dsn` → `DATABASE_DSN`）。

### 3. 启动服务

需要**两个终端**，API 和 Worker 同时运行：

```bash
# 终端 1 — API（默认 :8080）
go run ./cmd/api

# 终端 2 — Worker（链上监听 + 跟单）
go run ./cmd/worker
```

验证：访问 `http://localhost:8080/health`，应返回 `{"status":"ok"}`。

> 没有 Docker 也没关系。`docker-compose.yml` 仅在没有本地 MySQL 时可选使用。

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.port` | API 监听端口 | `8080` |
| `database.dsn` | MySQL 连接串 | 见 config.yaml |
| `auth.jwt_secret` | JWT 签名密钥 | 生产环境务必修改 |
| `auth.wallet_encrypt_key` | 跟单钱包私钥加密密钥（32 字节） | 生产环境务必修改 |
| `worker.poll_interval_sec` | 区块扫描间隔（秒） | `3` |
| `worker.confirmations` | 确认区块数（防重组） | `1` |
| `chains.bsc.enabled` | 是否启用 BSC | `true` |
| `chains.bsc.rpc_url` | BSC RPC 地址 | 公共节点 |
| `chains.ethereum.enabled` | 是否启用 Ethereum | `false` |

## API 接口

### 公开接口（无需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/api/meta/chains` | 获取支持的链与 DEX 列表 |
| POST | `/api/auth/nonce` | 获取登录 nonce 和待签名消息 |
| POST | `/api/auth/verify` | 提交签名，返回 JWT |

### 需登录接口（Header: `Authorization: Bearer <token>`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/me` | 当前用户信息 |
| GET | `/api/configs` | 跟单配置列表 |
| POST | `/api/configs` | 新增跟单配置 |
| PUT | `/api/configs/:id` | 更新跟单配置 |
| DELETE | `/api/configs/:id` | 删除跟单配置 |
| GET | `/api/wallets` | 跟单钱包列表 |
| POST | `/api/wallets` | 生成跟单钱包（每链一个） |
| GET | `/api/copy-trades` | 我的跟单记录 |
| GET | `/api/leader-trades` | 监听到的领头交易 |

### 登录流程

```
前端                          后端
 │  POST /auth/nonce {address}
 │ ──────────────────────────▶  生成 nonce + SIWE 消息
 │ ◀──────────────────────────  { nonce, message }
 │  钱包 personal_sign(message)
 │  POST /auth/verify {address, message, signature}
 │ ──────────────────────────▶  验签 → 签发 JWT
 │ ◀──────────────────────────  { token }
```

## 扩展点

后续优化可在以下模块扩展，无需大改架构：

| 模块 | 可扩展方向 |
|------|-----------|
| `listener` | WebSocket 订阅、mempool 监听、第三方 Indexer |
| `strategy` | 代币白名单、蜜罐检测、每日限额 |
| `executor` | 消息队列、失败重试、nonce 管理、Gas 策略 |
| `dex` | Uniswap V3、PancakeSwap V3、聚合器 |
| `chain` | 更多 EVM 链、测试网 |
| `pkg/crypto` | KMS / HSM 替代本地 AES 加密 |

## 常见问题

**Q: API 启动报数据库连接失败？**  
检查 `database.dsn` 中的用户名、密码、库名是否正确，MySQL 服务是否已启动。

**Q: Worker 启动了但没有跟单？**  
确认：① 已添加并启用跟单配置；② 已创建跟单钱包且地址有 BNB；③ 领头地址在链上确实有 PancakeSwap V2 买入交易；④ BSC RPC 可正常访问。

**Q: 如何切换到测试网？**  
修改 `config.yaml` 中对应链的 `chain_id`、`rpc_url` 和 DEX 合约地址（当前默认为 BSC 主网）。
