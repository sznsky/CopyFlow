import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { apiClient } from '../api/client'
import type { WalletTrade } from '../types'

export function WalletDetail() {
  const { address } = useParams<{ address: string }>()
  const [trades, setTrades] = useState<WalletTrade[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (address) fetchTrades()
  }, [address])

  const fetchTrades = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await apiClient.get<{ trades: WalletTrade[] }>(`/api/wallet-history/${address}?limit=100`)
      setTrades(res.data.trades || [])
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } }; message?: string }
      setError(e.response?.data?.error || e.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const fmt = (d: string) => new Date(d).toLocaleString('zh-CN', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  })

  const pnlPositive = (v: string) => parseFloat(v) >= 0

  if (loading) {
    return (
      <div className="loading-state">
        <div className="spinner" />
        <span>加载钱包历史...</span>
      </div>
    )
  }

  return (
    <div className="page-wrap">
      {/* Back + title */}
      <Link to="/smart-wallets" className="back-link">← 返回聪明钱包</Link>

      <div className="page-header">
        <div>
          <h2>钱包详情</h2>
          <p className="detail-address">{address}</p>
        </div>
        <div className="stat-card" style={{ padding: '10px 16px', minWidth: 'auto' }}>
          <span className="stat-label">记录条数</span>
          <span className="stat-value" style={{ fontSize: 20 }}>{trades.length}</span>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="info-banner" style={{ borderColor: 'rgba(239,68,68,.3)', background: 'var(--danger-light)', color: 'var(--danger)' }}>
          {error}
        </div>
      )}

      {/* Trades table */}
      {trades.length === 0 ? (
        <div className="center-card">
          <span className="card-icon">📋</span>
          <h2>暂无交易历史</h2>
          <p>该钱包尚无可显示的交易记录</p>
        </div>
      ) : (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>时间</th>
                <th>方向</th>
                <th>DEX</th>
                <th>卖出</th>
                <th>买入</th>
                <th>金额 (USD)</th>
                <th>盈亏</th>
                <th>Tx</th>
              </tr>
            </thead>
            <tbody>
              {trades.map((trade) => (
                <tr key={trade.id}>
                  <td style={{ whiteSpace: 'nowrap', color: 'var(--muted)', fontSize: 12 }}>
                    {fmt(trade.block_time)}
                  </td>
                  <td>
                    <span className={`badge ${trade.is_buy ? 'badge-buy' : 'badge-sell'}`}>
                      {trade.is_buy ? '买入' : '卖出'}
                    </span>
                  </td>
                  <td style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
                    {trade.dex_name}
                  </td>
                  <td>
                    <div style={{ fontWeight: 600, fontSize: 13 }}>
                      {trade.token_in_symbol || '未知'}
                    </div>
                    <div className="mono">{parseFloat(trade.amount_in).toFixed(4)}</div>
                  </td>
                  <td>
                    <div style={{ fontWeight: 600, fontSize: 13 }}>
                      {trade.token_out_symbol || '未知'}
                    </div>
                    <div className="mono">{parseFloat(trade.amount_out).toFixed(4)}</div>
                  </td>
                  <td style={{ fontWeight: 600 }}>
                    ${parseFloat(trade.amount_usd).toLocaleString(undefined, { maximumFractionDigits: 2 })}
                  </td>
                  <td>
                    {trade.pnl_usd && trade.pnl_percent ? (
                      <>
                        <div className={pnlPositive(trade.pnl_usd) ? 'text-success' : 'text-danger'}
                          style={{ fontWeight: 600, fontSize: 13 }}>
                          {pnlPositive(trade.pnl_usd) ? '+' : ''}
                          ${parseFloat(trade.pnl_usd).toLocaleString(undefined, { maximumFractionDigits: 2 })}
                        </div>
                        <div className={`mono ${pnlPositive(trade.pnl_percent) ? 'text-success' : 'text-danger'}`}>
                          ({pnlPositive(trade.pnl_percent) ? '+' : ''}{parseFloat(trade.pnl_percent).toFixed(2)}%)
                        </div>
                      </>
                    ) : (
                      <span style={{ color: 'var(--muted)', fontSize: 13 }}>—</span>
                    )}
                  </td>
                  <td>
                    <a
                      href={`https://etherscan.io/tx/${trade.tx_hash}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="mono"
                    >
                      {trade.tx_hash.slice(0, 6)}...{trade.tx_hash.slice(-4)}
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
