import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createConfig, http, WagmiProvider } from 'wagmi'
import { bsc, mainnet } from 'wagmi/chains'
import App from './App'
import './index.css'

// Wagmi 配置：支持 Ethereum + BSC
// 使用默认 localStorage 持久化，刷新后自动静默恢复连接状态（eth_accounts），
// 不会弹出钱包授权窗；连接状态与登录态（JWT）都会在刷新后保持。
const wagmiConfig = createConfig({
  chains: [bsc, mainnet],
  transports: {
    [bsc.id]: http(),
    [mainnet.id]: http(),
  },
})

const queryClient = new QueryClient()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <WagmiProvider config={wagmiConfig}>
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </WagmiProvider>
  </StrictMode>,
)
