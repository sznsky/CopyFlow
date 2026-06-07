import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createConfig, http, WagmiProvider } from 'wagmi'
import { bsc, mainnet } from 'wagmi/chains'
import App from './App'
import './index.css'

// Wagmi 配置：支持 Ethereum + BSC
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
