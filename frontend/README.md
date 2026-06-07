# CopyFlow Frontend

CopyFlow 链上跟单平台的前端界面，基于 React + Vite 构建，通过 MetaMask 钱包签名登录，管理跟单配置并查看交易记录。

## 技术栈

| 技术 | 用途 |
|------|------|
| React 18 | UI 框架 |
| TypeScript | 类型安全 |
| Vite 5 | 构建与开发服务器 |
| wagmi + viem | 钱包连接与链交互 |
| react-router-dom | 页面路由 |
| @tanstack/react-query | 异步状态管理（wagmi 依赖） |

## 功能页面

| 路由 | 页面 | 功能 |
|------|------|------|
| `/` | 概览 | 链支持情况、活跃配置数、最近跟单记录 |
| `/configs` | 跟单配置 | 新增 / 启用 / 停用 / 删除领头地址跟单策略 |
| `/wallets` | 跟单钱包 | 按链生成专用钱包，展示充值地址 |
| `/trades` | 交易记录 | 我的跟单 + 领头交易，支持自动刷新和区块浏览器链接 |

## 目录结构

```
frontend/
├── src/
│   ├── api/
│   │   └── client.ts         # API 请求封装（JWT 自动携带）
│   ├── components/
│   │   ├── Layout.tsx        # 顶栏导航 + 页面布局
│   │   └── ConnectWallet.tsx # 钱包连接 + SIWE 签名登录
│   ├── context/
│   │   └── AuthContext.tsx   # 全局登录态（user / login / logout）
│   ├── hooks/
│   │   └── useInterval.ts    # 定时刷新 Hook
│   ├── pages/
│   │   ├── Dashboard.tsx     # 首页概览
│   │   ├── Configs.tsx       # 跟单配置管理
│   │   ├── Wallets.tsx       # 跟单钱包管理
│   │   └── Trades.tsx        # 交易记录
│   ├── utils/
│   │   └── explorer.ts       # 区块浏览器链接（BscScan / Etherscan）
│   ├── types.ts              # 后端响应类型定义
│   ├── App.tsx               # 路由配置
│   ├── main.tsx              # 入口（wagmi + QueryClient 初始化）
│   └── index.css             # 全局样式（暗色主题）
├── vite.config.ts            # Vite 配置（含 API 代理）
└── package.json
```

## 环境要求

- Node.js 18+
- 后端 API 已启动（默认 `http://localhost:8080`）
- 浏览器安装 MetaMask 插件

## 快速开始

```bash
# 安装依赖
npm install

# 启动开发服务器（默认 http://localhost:5173）
npm run dev

# 生产构建
npm run build

# 预览构建产物
npm run preview
```

**启动前请确保后端 API 已运行**，否则登录和接口请求会失败。

## 使用流程

```
1. 连接钱包（MetaMask）
       ↓
2. 签名登录（SIWE）
       ↓
3. 创建跟单钱包（/wallets）→ 向地址充值 BNB
       ↓
4. 添加跟单配置（/configs）→ 填写领头地址、链、比例等
       ↓
5. 等待 Worker 监听到领头买入 → 自动跟单
       ↓
6. 在交易记录（/trades）查看状态
```

### 跟单配置说明

| 字段 | 说明 | 示例 |
|------|------|------|
| 链 | 当前 MVP 主要支持 BSC | BSC (56) |
| DEX | 去中心化交易所类型 | pancake_v2 |
| 领头地址 | 要跟单的目标钱包地址 | `0x...` |
| 跟单模式 | 等比例 或 固定金额 | 比例 `0.1` = 10% |
| 单笔上限 | 单次跟单最大 BNB 数量 | `1` |
| 滑点 | 基点，300 = 3% | `300` |

## 开发配置

### API 代理

开发环境下，`vite.config.ts` 将 `/api` 和 `/health` 代理到后端：

```ts
proxy: {
  '/api': { target: 'http://localhost:8080' },
  '/health': { target: 'http://localhost:8080' },
}
```

前端请求使用相对路径（如 `/api/configs`），无需硬编码后端地址。若后端端口不同，修改 `vite.config.ts` 中的 `target`。

### 支持的链

`main.tsx` 中 wagmi 配置了 BSC 和 Ethereum：

```ts
chains: [bsc, mainnet]
```

钱包连接时用户可选择对应网络。实际跟单能力取决于后端 `config.yaml` 中启用的链。

### 登录态

- JWT 存储在 `localStorage`（key: `copyflow_token`）
- 刷新页面后自动调用 `/api/me` 恢复登录态
- 点击「退出」清除 token

## 常见问题

**Q: 点击「签名登录」没反应？**  
确认 MetaMask 已安装、已连接正确网络，且后端 API 正常运行。

**Q: 接口报 401？**  
Token 可能已过期，退出后重新签名登录。

**Q: 交易记录不更新？**  
交易记录页每 10 秒自动刷新；也可点击「刷新」按钮。跟单由后端 Worker 执行，需确认 Worker 进程已启动。

**Q: 如何修改后端地址？**  
开发环境改 `vite.config.ts` 的 proxy；生产环境需配置 Nginx 反向代理或修改 API base URL。
