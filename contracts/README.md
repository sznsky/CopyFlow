# CopyFlow Contracts

智能合约目录，当前 **MVP 阶段为空实现**。

MVP 通过 Go 后端直接调用链上已有 DEX Router（PancakeSwap / Uniswap）完成跟单，不部署自研合约。

## 后续可扩展方向

| 合约 | 用途 |
|------|------|
| `CopyVault.sol` | 用户资金托管金库，替代服务端存私钥 |
| `CopyRouter.sol` | 统一跟单入口，聚合多 DEX |
| `CopyRegistry.sol` | 链上记录跟单关系，可审计 |

## 推荐工具链

- [Foundry](https://book.getfoundry.sh/) 或 [Hardhat](https://hardhat.org/)
- 测试网：BSC Testnet / Sepolia

## 初始化 Foundry（可选）

```bash
cd contracts
forge init --no-commit .
```

初始化后可将接口文件放在 `src/interfaces/` 下。
