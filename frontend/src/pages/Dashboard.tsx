import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'
import type { ChainMeta, CopyConfig, CopyTrade } from '../types'

/** 首页概览：展示链支持、配置数、最近跟单 */
export function Dashboard() {
  const { user } = useAuth()
  const [chains, setChains] = useState<ChainMeta[]>([])
  const [configs, setConfigs] = useState<CopyConfig[]>([])
  const [trades, setTrades] = useState<CopyTrade[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user) return
    Promise.all([api.fetchChains(), api.fetchConfigs(), api.fetchCopyTrades()])
      .then(([c, cfg, t]) => {
        setChains(c)
        setConfigs(cfg)
        setTrades(t.slice(0, 5))
      })
      .finally(() => setLoading(false))
  }, [user])

  if (!user) {
    return (
      <div className="card center-card">
        <h2>欢迎使用 CopyFlow</h2>
        <p className="text-muted">请先连接钱包并签名登录，开始配置链上跟单。</p>
      </div>
    )
  }

  if (loading) return <p className="text-muted">加载中...</p>

  const activeConfigs = configs.filter((c) => c.is_active).length

  return (
    <div className="page">
      <h2>概览</h2>
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">活跃跟单配置</span>
          <span className="stat-value">{activeConfigs}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">总配置数</span>
          <span className="stat-value">{configs.length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">支持链</span>
          <span className="stat-value">{chains.filter((c) => c.enabled).length}</span>
        </div>
      </div>

      <section className="section">
        <h3>支持的链</h3>
        <div className="chain-list">
          {chains.map((ch) => (
            <div key={ch.chain_id} className={`chain-chip ${ch.enabled ? '' : 'disabled'}`}>
              <strong>{ch.name}</strong>
              <span>{ch.enabled ? '已启用' : '未启用'}</span>
              <span className="text-muted">
                {ch.dexes.filter((d) => d.enabled).map((d) => d.type).join(', ') || '无 DEX'}
              </span>
            </div>
          ))}
        </div>
      </section>

      <section className="section">
        <div className="section-header">
          <h3>最近跟单</h3>
          <Link to="/trades">查看全部</Link>
        </div>
        {trades.length === 0 ? (
          <p className="text-muted">暂无跟单记录</p>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>状态</th>
                <th>代币</th>
                <th>金额</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              {trades.map((t) => (
                <tr key={t.id}>
                  <td><StatusBadge status={t.status} /></td>
                  <td className="mono">{shortAddr(t.token_out)}</td>
                  <td>{t.amount_in}</td>
                  <td>{new Date(t.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <div className="quick-actions">
        <Link className="btn btn-primary" to="/configs">添加跟单配置</Link>
        <Link className="btn btn-secondary" to="/wallets">管理跟单钱包</Link>
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const cls = `badge badge-${status}`
  const labels: Record<string, string> = {
    pending: '待处理',
    submitted: '已提交',
    success: '成功',
    failed: '失败',
    skipped: '跳过',
  }
  return <span className={cls}>{labels[status] || status}</span>
}

function shortAddr(addr: string) {
  return `${addr.slice(0, 6)}...${addr.slice(-4)}`
}
