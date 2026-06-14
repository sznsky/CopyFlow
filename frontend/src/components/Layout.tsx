import { Link, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { AuthButtons } from './AuthButtons'
import { Logo } from './Logo'

const navItems = [
  { path: '/', label: '概览' },
  { path: '/configs', label: '跟单配置' },
  { path: '/wallets', label: '跟单钱包' },
  { path: '/trades', label: '交易记录' },
  { path: '/smart-wallets', label: '聪明钱包' },
  { path: '/token-signals', label: '代币信号' },
]

/** 应用主布局：顶栏导航 + 内容区 */
export function Layout() {
  const { user, logout } = useAuth()
  const location = useLocation()

  return (
    <div className="app">
      <header className="header">
        <div className="header-inner">
          <Logo />
          <nav className="nav">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={location.pathname === item.path ? 'nav-link active' : 'nav-link'}
              >
                {item.label}
              </Link>
            ))}
          </nav>
          <div className="header-actions">
            {user ? (
              <>
                <span className="wallet-badge" title={user.email || user.wallet_address}>
                  {user.email ||
                    (user.wallet_address
                      ? `${user.wallet_address.slice(0, 6)}...${user.wallet_address.slice(-4)}`
                      : '用户')}
                </span>
                <button className="btn btn-ghost" onClick={logout}>退出</button>
              </>
            ) : (
              <AuthButtons />
            )}
          </div>
        </div>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
