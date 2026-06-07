import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'
import { useInterval } from '../hooks/useInterval'
import type { CopyTrade, LeaderTrade } from '../types'
import { chainName, txExplorerUrl } from '../utils/explorer'

/** 交易记录页：领头交易 + 我的跟单（每 10 秒自动刷新） */
export function Trades() {
  const { user } = useAuth()
  const [tab, setTab] = useState<'copy' | 'leader'>('copy')
  const [copyTrades, setCopyTrades] = useState<CopyTrade[]>([])
  const [leaderTrades, setLeaderTrades] = useState<LeaderTrade[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    const [copy, leader] = await Promise.all([api.fetchCopyTrades(), api.fetchLeaderTrades()])
    setCopyTrades(copy)
    setLeaderTrades(leader)
    setLoading(false)
  }, [])

  useEffect(() => {
    if (!user) return
    load()
  }, [user, load])

  // 自动刷新，跟踪 submitted 状态变化
  useInterval(() => {
    if (user) load()
  }, user ? 10_000 : null)

  if (!user) return <p className="text-muted">请先登录</p>
  if (loading) return <p className="text-muted">加载中...</p>

  return (
    <div className="page">
      <div className="section-header">
        <h2>交易记录</h2>
        <button className="btn btn-ghost" onClick={load}>刷新</button>
      </div>
      <div className="tabs">
        <button className={tab === 'copy' ? 'tab active' : 'tab'} onClick={() => setTab('copy')}>
          我的跟单 ({copyTrades.length})
        </button>
        <button className={tab === 'leader' ? 'tab active' : 'tab'} onClick={() => setTab('leader')}>
          领头交易 ({leaderTrades.length})
        </button>
      </div>

      {tab === 'copy' ? (
        <table className="table">
          <thead>
            <tr>
              <th>状态</th>
              <th>链</th>
              <th>代币</th>
              <th>金额 (wei)</th>
              <th>Tx Hash</th>
              <th>错误</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {copyTrades.length === 0 ? (
              <tr><td colSpan={7} className="text-muted">暂无跟单记录</td></tr>
            ) : (
              copyTrades.map((t) => {
                const chainId = t.leader_trade?.chain_id ?? 56
                return (
                  <tr key={t.id}>
                    <td><StatusBadge status={t.status} /></td>
                    <td>{chainName(chainId)}</td>
                    <td className="mono">{shortAddr(t.token_out)}</td>
                    <td>{t.amount_in}</td>
                    <td>
                      <TxLink chainId={chainId} hash={t.tx_hash} />
                    </td>
                    <td className="error-text">{t.error_msg || '-'}</td>
                    <td>{new Date(t.created_at).toLocaleString()}</td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>链</th>
              <th>领头地址</th>
              <th>DEX</th>
              <th>买入代币</th>
              <th>Tx Hash</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {leaderTrades.length === 0 ? (
              <tr><td colSpan={6} className="text-muted">暂无领头交易</td></tr>
            ) : (
              leaderTrades.map((t) => (
                <tr key={t.id}>
                  <td>{chainName(t.chain_id)}</td>
                  <td className="mono">{shortAddr(t.leader_address)}</td>
                  <td>{t.dex_type}</td>
                  <td className="mono">{shortAddr(t.token_out)}</td>
                  <td><TxLink chainId={t.chain_id} hash={t.tx_hash} /></td>
                  <td>{new Date(t.detected_at).toLocaleString()}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      )}
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const labels: Record<string, string> = {
    pending: '待处理',
    submitted: '已提交',
    success: '成功',
    failed: '失败',
    skipped: '跳过',
  }
  return <span className={`badge badge-${status}`}>{labels[status] || status}</span>
}

function TxLink({ chainId, hash }: { chainId: number; hash: string | null }) {
  if (!hash) return <span>-</span>
  const url = txExplorerUrl(chainId, hash)
  if (!url) return <span className="mono">{shortAddr(hash)}</span>
  return (
    <a href={url} target="_blank" rel="noreferrer" className="mono">
      {shortAddr(hash)}
    </a>
  )
}

function shortAddr(addr: string) {
  if (!addr || addr.length < 10) return addr
  return `${addr.slice(0, 6)}...${addr.slice(-4)}`
}
