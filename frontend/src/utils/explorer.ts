/** 根据 chainId 生成区块浏览器交易链接 */

const explorers: Record<number, string> = {
  1: 'https://etherscan.io/tx/',
  56: 'https://bscscan.com/tx/',
  97: 'https://testnet.bscscan.com/tx/',
  11155111: 'https://sepolia.etherscan.io/tx/',
}

/** 获取交易哈希的浏览器链接，未知链返回 null */
export function txExplorerUrl(chainId: number, txHash: string): string | null {
  const base = explorers[chainId]
  if (!base || !txHash) return null
  return `${base}${txHash}`
}

/** 链 ID 到显示名称 */
export function chainName(chainId: number): string {
  const names: Record<number, string> = {
    1: 'Ethereum',
    56: 'BSC',
    97: 'BSC Testnet',
    11155111: 'Sepolia',
  }
  return names[chainId] || `Chain ${chainId}`
}
