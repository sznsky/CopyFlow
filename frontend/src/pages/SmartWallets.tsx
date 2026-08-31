import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '../api/client'
import type { SmartWallet } from '../types'

export function SmartWallets() {
  const [wallets, setWallets] = useState<SmartWallet[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => { fetchWallets() }, [])

  const fetchWallets = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await apiClient.get('/api/smart-wallets?limit=20&min_score=60')
      setWallets(res.data.wallets || [])
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } }; message?: string }
      setError(e.response?.data?.error || e.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="loading-state">
        <div className="spinner" />
        <span>加载聪明钱包...</span>
      </div>
    )
  }

  return (
    <div className="page-wrap">
      {/* Back */}
      <Link to="/" className="back-link">← 返回首页</Link>

      {/* Header */}
      <div className="page-header">
        <div>
          <h2>聪明钱包</h2>
          <p className="page-desc">按综合评分排名的 Top 20 优质钱包</p>
        </div>
      </div>

      {/* Summary stats */}
      <div className="stats-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))' }}>
        <div className="stat-card">
          <span className="stat-label">钱包总数</span>
          <span className="stat-value">{wallets.length}</span>
        </div>
        <div className="stat-card stat-brand">
          <span className="stat-label">评估周期</span>
          <span className="stat-value" style={{ fontSize: 16, paddingTop: 4 }}>近 6 月</span>
        </div>
      </div>

      {/* Error */}
      {error && <div className="info-banner" style={{ borderColor: 'rgba(239,68,68,.3)', background: 'var(--danger-light)', color: 'var(--danger)' }}>{error}</div>}

      {/* Wallet cards */}
      {wallets.length === 0 ? (
        <div className="center-card">
          <span className="card-icon">🔍</span>
          <h2>暂无聪明钱包数据</h2>
          <p>评分 60 分以上的钱包将显示在这里</p>
        </div>
      ) : (
        <div className="wallet-cards-grid">
          {wallets.map((wallet) => {
            const score = parseFloat(wallet.score)
            const pnl = parseFloat(wallet.total_pnl)
            const winRate = parseFloat(wallet.win_rate)

            return (
              <Link
                key={wallet.id}
                to={`/smart-wallets/${wallet.wallet_address}`}
                style={{ textDecoration: 'none' }}
              >
                <div className="smart-wallet-card">
                  {/* Card header: rank + address */}
                  <div className="swc-header">
                    <span className="swc-rank">#{wallet.rank_position ?? '-'}</span>
                    <span className="swc-address">
                      {wallet.wallet_address.slice(0, 6)}...{wallet.wallet_address.slice(-4)}
                    </span>
                  </div>

                  {/* Score row */}
                  <div className="swc-score-row">
                    <span className="swc-score-label">综合评分</span>
                    <span className={`score-pill ${score >= 80 ? 'score-high' : score >= 70 ? 'score-mid' : 'score-low'}`}>
                      {score.toFixed(1)}
                    </span>
                  </div>
                  <div className="progress-bar">
                    <div
                      className={`progress-fill ${score >= 80 ? 'fill-success' : ''}`}
                      style={{ width: `${Math.min(score, 100)}%` }}
                    />
                  </div>

                  {/* Metrics grid */}
                  <div className="swc-metrics">
                    <div className="swc-metric">
                      <span className="swc-metric-label">累计盈亏</span>
                      <span className={`swc-metric-value ${pnl >= 0 ? 'text-success' : 'text-danger'}`}>
                        {pnl >= 0 ? '+' : ''}${pnl.toLocaleString(undefined, { maximumFractionDigits: 0 })}
                      </span>
                    </div>
                    <div className="swc-metric">
                      <span className="swc-metric-label">胜率</span>
                      <span className="swc-metric-value">{winRate.toFixed(1)}%</span>
                    </div>
                    <div className="swc-metric">
                      <span className="swc-metric-label">总交易</span>
                      <span className="swc-metric-value">{wallet.total_trades}</span>
                    </div>
                    <div className="swc-metric">
                      <span className="swc-metric-label">盈亏比</span>
                      <span className="swc-metric-value">{parseFloat(wallet.profit_loss_ratio).toFixed(2)}</span>
                    </div>
                  </div>

                  <div style={{ textAlign: 'right' }}>
                    <span className="btn btn-sm btn-primary">查看详情 →</span>
                  </div>
                </div>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}
