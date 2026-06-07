import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { AuthProvider } from './context/AuthContext'
import { Configs } from './pages/Configs'
import { Dashboard } from './pages/Dashboard'
import { Trades } from './pages/Trades'
import { Wallets } from './pages/Wallets'

/** 应用根组件：路由 + 全局 Auth */
export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="configs" element={<Configs />} />
            <Route path="wallets" element={<Wallets />} />
            <Route path="trades" element={<Trades />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
