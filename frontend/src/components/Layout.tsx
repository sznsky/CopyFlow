import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation } from 'react-router-dom'
import { useDisconnect } from 'wagmi'
import { useAuth } from '../context/AuthContext'
import { ConnectWallet } from './ConnectWallet'
import { Logo } from './Logo'

const navItems = [
  { path: '/', label: '概览' },
  { path: '/configs', label: '跟单配置' },
  { path: '/wallets', label: '跟单钱包' },
  { path: '/trades', label: '交易记录' },
  { path: '/smart-wallets', label: '聪明钱包' },
  // { path: '/token-signals', label: '代币信号' },
]

export function Layout() {
  const { user, logout } = useAuth()
  const { disconnect } = useDisconnect()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)

  // 退出：同时清除 JWT 和断开 wagmi 钱包，防止 ConnectWallet 自动重新签名
  const handleLogout = () => {
    logout()
    disconnect()
  }

  useEffect(() => { setMenuOpen(false) }, [location.pathname])

  useEffect(() => {
    document.body.style.overflow = menuOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [menuOpen])

  const isActive = (path: string) =>
    path === '/' ? location.pathname === '/' : location.pathname.startsWith(path)

  return (
    <div className="app">
      <div className="copyflow-container">

        {/* ── Navbar ── */}
        <nav className="navbar">
          {/* 顶行：Logo 左，用户操作 右 */}
          <div className="navbar-top">
            <div className="brand">
              <Logo size={48} />
              <div className="tagline">
                发现聪明钱正在买什么
              </div>
            </div>

            {/* Desktop: 已登录显示用户标识+退出，未登录显示连接钱包 */}
            <div className="navbar-actions">
              {user ? (
                <>
                  <span className="user-badge" title={user.wallet_address || user.email || ''}>
                    <i className="fas fa-wallet" style={{ marginRight: 5, fontSize: 10 }} />
                    {user.email ||
                      (user.wallet_address
                        ? `0x...${user.wallet_address.slice(-4)}`
                        : '用户')}
                  </span>
                  <button className="logout-btn" onClick={handleLogout}>
                    <i className="fas fa-right-from-bracket" style={{ marginRight: 4 }} />
                    退出
                  </button>
                </>
              ) : (
                <ConnectWallet />
              )}
            </div>

            {/* Hamburger (mobile only) */}
            <button
              className={`hamburger ${menuOpen ? 'open' : ''}`}
              onClick={() => setMenuOpen((v) => !v)}
              aria-label={menuOpen ? '关闭菜单' : '打开菜单'}
            >
              <span className="hamburger-line" />
              <span className="hamburger-line" />
              <span className="hamburger-line" />
            </button>
          </div>

          {/* 底行：导航链接 */}
          <div className="nav-links">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={isActive(item.path) ? 'nav-link active' : 'nav-link'}
              >
                {item.label}
              </Link>
            ))}
          </div>
        </nav>

        {/* ── Mobile drawer ── */}
        {menuOpen && (
          <div
            className="mobile-drawer-overlay open"
            onClick={() => setMenuOpen(false)}
          >
            <div className="mobile-drawer" onClick={(e) => e.stopPropagation()}>
              {/* Drawer header */}
              <div className="mobile-drawer-header">
                <Logo size={36} />
                <button
                  onClick={() => setMenuOpen(false)}
                  style={{
                    background: 'none', border: 'none', fontSize: 18,
                    cursor: 'pointer', color: 'var(--text-tip)', lineHeight: 1,
                  }}
                >
                  ✕
                </button>
              </div>

              {/* Nav links */}
              <div className="mobile-nav-links">
                {navItems.map((item) => (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={isActive(item.path) ? 'nav-link active' : 'nav-link'}
                  >
                    {item.label}
                  </Link>
                ))}
              </div>

              {/* Mobile user area */}
              <div className="mobile-user-area">
                {user ? (
                  <>
                    <span className="user-badge" style={{ flex: 1 }}
                      title={user.wallet_address || user.email || ''}>
                      <i className="fas fa-wallet" style={{ marginRight: 5, fontSize: 10 }} />
                      {user.email ||
                        (user.wallet_address
                          ? `0x...${user.wallet_address.slice(-4)}`
                          : '用户')}
                    </span>
                    <button className="logout-btn" onClick={handleLogout}>退出</button>
                  </>
                ) : (
                  <ConnectWallet />
                )}
              </div>
            </div>
          </div>
        )}

        {/* ── Page content ── */}
        <main className="main-content">
          <Outlet />
        </main>

        {/* ── Footer ── */}
        <div className="footer-note">
          <i className="fas fa-database" />
          数据源：链上聚合 · 实时同步聪明钱交易
        </div>

      </div>
    </div>
  )
}
