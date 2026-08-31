import { useEffect, useState } from 'react'
import { apiClient } from '../api/client'
import type { TokenSignal } from '../types'

export function TokenSignals() {
  const [signals, setSignals] = useState<TokenSignal[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => { fetchSignals() }, [])

  const fetchSignals = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await apiClient.get('/api/token-signals?limit=20&min_consensus_score=50')
      setSignals(res.data.signals || [])
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } }; message?: string }
      setError(e.response?.data?.error || e.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const fmt = (dateStr: string) =>
    new Date(dateStr).toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })

  const scoreClass = (s: string) => {
    const v = parseFloat(s)
    return v >= 80 ? 'score-high' : v >= 60 ? 'score-mid' : 'score-low'
  }

  if (loading) {
    return (
      <div className="loading-state">
        <div className="spinner" />
        <span>加载代币信号...</span>
      </div>
    )
  }

  return (
    <div className="page-wrap">
      {/* Header */}
      <div className="page-header">
        <div>
          <h2>代币信号</h2>
          <p className="page-desc">近 3 天内被多个聪明钱包买入的代币</p>
        </div>
        <button className="btn btn-secondary" onClick={fetchSignals}>刷新</button>
      </div>

      {/* Summary */}
      <div className="stats-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))' }}>
        <div className="stat-card">
          <span className="stat-label">信号总数</span>
          <span className="stat-value">{signals.length}</span>
        </div>
        <div className="stat-card stat-brand">
          <span className="stat-label">统计周期</span>
          <span className="stat-value" style={{ fontSize: 16, paddingTop: 4 }}>近 3 天</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">最低共识分</span>
          <span className="stat-value">50</span>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="info-banner" style={{ borderColor: 'rgba(239,68,68,.3)', background: 'var(--danger-light)', color: 'var(--danger)' }}>
          {error}
        </div>
      )}

      {/* Signal cards */}
      {signals.length === 0 ? (
        <div className="center-card">
          <span className="card-icon">📡</span>
          <h2>暂无代币信号</h2>
          <p>共识分 50 分以上的代币信号将显示在这里</p>
        </div>
      ) : (
        <div className="signal-cards">
          {signals.map((signal) => (
            <div key={signal.id} className="signal-card">
              {/* Token header */}
              <div className="signal-header">
                <div className="signal-title-group">
                  <div className="signal-token-name">
                    {signal.token_symbol || '未知代币'}
                    {signal.token_name && (
                      <span className="text-muted" style={{ fontSize: 13, fontWeight: 400 }}>
                        {signal.token_name}
                      </span>
                    )}
                    <span className={`score-pill ${scoreClass(signal.consensus_score)}`}>
                      共识 {parseFloat(signal.consensus_score).toFixed(1)}
                    </span>
                  </div>
                  <div className="signal-address">{signal.token_address}</div>
                </div>
              </div>

              {/* Metrics */}
              <div className="signal-metrics">
                <div>
                  <div className="signal-metric-label">聪明钱包数</div>
                  <div className="signal-metric-value">{signal.smart_wallet_count}</div>
                </div>
                <div>
                  <div className="signal-metric-label">总买入量</div>
                  <div className="signal-metric-value">
                    ${parseFloat(signal.total_buy_volume).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                  </div>
                </div>
                <div>
                  <div className="signal-metric-label">平均买入</div>
                  <div className="signal-metric-value">
                    ${parseFloat(signal.avg_buy_amount).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                  </div>
                </div>
                <div>
                  <div className="signal-metric-label">首次买入</div>
                  <div className="signal-metric-value" style={{ fontSize: 13 }}>{fmt(signal.first_buy_time)}</div>
                </div>
              </div>

              {/* Footer: price / market cap / liquidity */}
              {(signal.price_usd || signal.market_cap || signal.liquidity_usd) && (
                <div className="signal-footer">
                  {signal.price_usd && (
                    <div className="signal-footer-item">
                      <span className="signal-footer-label">价格</span>
                      <span className="signal-footer-value">${parseFloat(signal.price_usd).toFixed(6)}</span>
                    </div>
                  )}
                  {signal.market_cap && (
                    <div className="signal-footer-item">
                      <span className="signal-footer-label">市值</span>
                      <span className="signal-footer-value">
                        ${parseFloat(signal.market_cap).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                      </span>
                    </div>
                  )}
                  {signal.liquidity_usd && (
                    <div className="signal-footer-item">
                      <span className="signal-footer-label">流动性</span>
                      <span className="signal-footer-value">
                        ${parseFloat(signal.liquidity_usd).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                      </span>
                    </div>
                  )}
                  <div className="signal-footer-item">
                    <span className="signal-footer-label">最近买入</span>
                    <span className="signal-footer-value">{fmt(signal.last_buy_time)}</span>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
