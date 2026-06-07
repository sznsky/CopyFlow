# CopyFlow

链上跟单平台：监听指定领头地址的 DEX 买入交易，按策略自动跟单。

## 项目结构

```
CopyFlow/
├── backend/      # Go 后端（API + Worker）  → 详见 backend/README.md
├── frontend/     # React 前端              → 详见 frontend/README.md
├── contracts/    # 智能合约（预留扩展）     → 详见 contracts/README.md
├── scripts/      # 开发启动脚本
└── remark.md     # 原始需求说明
```

## 5 分钟上手

**前置条件：** 本地 MySQL、Go 1.22+、Node.js 18+、MetaMask

```bash
# 1. 建库
mysql -u root -p -e "CREATE DATABASE copyflow DEFAULT CHARSET utf8mb4;"
mysql -u root -p copyflow < backend/migrations/001_init.sql

# 2. 改数据库连接（backend/config/config.yaml → database.dsn）

# 3. 启动后端（两个终端）
cd backend && go run ./cmd/api
cd backend && go run ./cmd/worker

# 4. 启动前端
cd frontend && npm install && npm run dev
```

浏览器打开 `http://localhost:5173`，连接钱包 → 签名登录 → 创建跟单钱包 → 添加配置。

## 文档索引

| 文档 | 内容 |
|------|------|
| [backend/README.md](backend/README.md) | 后端架构、配置、API 接口、扩展点 |
| [frontend/README.md](frontend/README.md) | 前端页面、目录结构、使用流程 |
| [contracts/README.md](contracts/README.md) | 合约扩展规划 |
| [remark.md](remark.md) | 项目需求与流程图 |

## 其他

- Windows 一键启动：`.\scripts\dev.ps1`（需先配好 MySQL）
- Makefile 快捷命令：`make api` / `make worker` / `make frontend`
