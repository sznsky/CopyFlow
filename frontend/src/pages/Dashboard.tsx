import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '../api/client'
import { useAuth } from '../context/AuthContext'
import { ConnectWallet } from '../components/ConnectWallet'
import type { SmartWallet, TokenSignal, WalletTrade } from '../types'

// ── Types ──────────────────────────────────────
interface ActivityItem {
  id: string
  walletRank: number
  walletAddress: string
  action: 'buy' | 'sell'
  token: string
  amountUsd: number
  timeLabel: string
  rawTime: number
}

// ── Helpers ────────────────────────────────────
function relativeTime(isoStr: string): string {
  try {
    const ms = Date.now() - new Date(isoStr).getTime()
    const mins = Math.floor(ms / 60000)
    if (mins < 1) return '刚刚'
    if (mins < 60) return `${mins}分钟前`
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return `${hrs}小时前`
    return `${Math.floor(hrs / 24)}天前`
  } catch {
    return '最新'
  }
}

function formatUsd(n: number): string {
  if (n >= 1_000_000) return `$${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1000) return `$${(n / 1000).toFixed(1)}K`
  return `$${n.toFixed(0)}`
}

// ── Toast ──────────────────────────────────────
function useToast() {
  const [msg, setMsg] = useState('')
  const timer = useRef<ReturnType<typeof setTimeout>>()
  const show = useCallback((text: string) => {
    setMsg(text)
    clearTimeout(timer.current)
    timer.current = setTimeout(() => setMsg(''), 2800)
  }, [])
  return { msg, show }
}

// ── Constants ──────────────────────────────────
const RANK_MEDALS = ['🥇', '🥈', '🥉']

// ── Main Dashboard ────────────────────────────
export function Dashboard() {
  const { user } = useAuth()
  const { msg: toast, show: showToast } = useToast()

  const [opportunities, setOpportunities] = useState<TokenSignal[]>([])
  const [oppLoading, setOppLoading] = useState(true)

  const [topWallets, setTopWallets] = useState<SmartWallet[]>([])
  const [walletsLoading, setWalletsLoading] = useState(true)

  const [activities, setActivities] = useState<ActivityItem[]>([])
  const [activitiesLoading, setActivitiesLoading] = useState(true)

  const [stats, setStats] = useState<{
    monitored_wallets: number
    today_signals: number
    top_score: number
    avg_win_rate: number
  } | null>(null)

  // Load dashboard stats
  useEffect(() => {
    apiClient
      .get('/api/dashboard/stats')
      .then((res) => setStats(res.data))
      .catch(() => setStats(null))
  }, [])

  // Load token signals
  useEffect(() => {
    apiClient
      .get('/api/token-signals?limit=3&min_consensus_score=20')
      .then((res) => setOpportunities(res.data.signals || []))
      .catch(() => setOpportunities([]))
      .finally(() => setOppLoading(false))
  }, [])

  // Load top wallets for leaderboard (by score)
  useEffect(() => {
    let active = true
    apiClient
      .get('/api/smart-wallets?limit=5&min_score=60')
      .then((res) => {
        if (!active) return
        setTopWallets(res.data.wallets || [])
      })
      .catch(() => { if (active) setTopWallets([]) })
      .finally(() => { if (active) setWalletsLoading(false) })
    return () => { active = false }
  }, [])

  // Load recent activity directly — sorted by block_time DESC across ALL qualifying wallets
  // (avoids stale data from high-scored but inactive wallets)
  useEffect(() => {
    let active = true
    apiClient
      .get('/api/recent-activity?limit=8&min_score=60')
      .then((res) => {
        if (!active) return
        const raw: Array<{
          wallet_address: string
          rank_position: number
          is_buy: boolean
          token_in: string
          token_out: string
          token_in_symbol: string
          token_out_symbol: string
          amount_usd: string
          block_time: string
          tx_hash: string
        }> = res.data.activities || []

        const items: ActivityItem[] = raw.map((r, idx) => {
          const sym = r.is_buy
            ? (r.token_out_symbol || r.token_out.slice(0, 6)).toUpperCase()
            : (r.token_in_symbol || r.token_in.slice(0, 6)).toUpperCase()
          return {
            id: r.tx_hash || String(idx),
            walletRank: r.rank_position ?? 0,
            walletAddress: r.wallet_address,
            action: r.is_buy ? 'buy' : 'sell',
            token: sym,
            amountUsd: parseFloat(r.amount_usd || '0'),
            timeLabel: relativeTime(r.block_time),
            rawTime: new Date(r.block_time).getTime(),
          }
        })
        setActivities(items)
      })
      .catch(() => { if (active) setActivities([]) })
      .finally(() => { if (active) setActivitiesLoading(false) })
    return () => { active = false }
  }, [])

  return (
    <>
      {/* ── Section 1: Platform Stats Bar ── */}
      <div className="stats-bar">
        {[
          { icon: 'fa-wallet',     label: '监控钱包', val: stats ? String(stats.monitored_wallets) : '0', cls: 'blue' },
          { icon: 'fa-fire',       label: '今日信号', val: stats ? String(stats.today_signals) : '0', cls: 'blue' },
          { icon: 'fa-trophy',     label: '最高评分', val: stats ? stats.top_score.toFixed(1) : '0.0', cls: 'blue' },
          { icon: 'fa-chart-line', label: '平均胜率', val: stats ? `${stats.avg_win_rate.toFixed(1)}%` : '0.0%', cls: 'blue' },
        ].map(({ icon, label, val, cls }) => (
          <div key={label} className={`stat-bar-item${cls ? ` stat-bar-${cls}` : ''}`}>
            <i className={`fas ${icon}`} />
            <div>
              <div className="stat-bar-value">{val}</div>
              <div className="stat-bar-label">{label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* ── Section 2 + 3: Opportunities + Activity ── */}
      <div className="dashboard-grid">
        <OpportunityCard
          opportunities={opportunities}
          loading={oppLoading}
          onOppClick={(sym, score) =>
            showToast(`智能分析：${sym} 综合评分 ${score}，多个聪明钱包共识买入`)
          }
        />
        <ActivityCard
          activities={activities}
          loading={activitiesLoading}
          onActivityClick={(act) =>
            showToast(
              `钱包 #${act.walletRank} ${act.action === 'buy' ? '买入' : '卖出'} ${act.token} ${act.amountUsd > 0 ? formatUsd(act.amountUsd) : ''}，可一键复制策略`
            )
          }
        />
      </div>

      {/* ── Section 4: Leaderboard Snapshot ── */}
      <LeaderboardSnapshot wallets={topWallets.slice(0, 3)} loading={walletsLoading} />

      {/* ── Section 5: CTA ── */}
      {!user && <GuestCTA />}

      {toast && <Toast msg={toast} />}
    </>
  )
}

// ── Section 2: Opportunity Card ─────────────────
function OpportunityCard({
  opportunities,
  loading,
  onOppClick,
}: {
  opportunities: TokenSignal[]
  loading: boolean
  onOppClick: (sym: string, score: number) => void
}) {
  return (
    <div className="card">
      <div className="card-header">
        <i className="fas fa-fire" />
        <h2>今日机会</h2>
        {opportunities.length > 0 && (
          <span className="card-header-actions">
            <Link to="/token-signals" className="btn btn-sm btn-ghost">全部 →</Link>
          </span>
        )}
      </div>
      <div className="opportunity-list">
        {loading ? (
          <div className="loading-state" style={{ padding: 32 }}>
            <div className="spinner" />
            <span>加载中...</span>
          </div>
        ) : opportunities.length > 0 ? (
          opportunities.map((sig, idx) => {
            const score = Math.round(parseFloat(sig.consensus_score))
            const sym = sig.token_symbol || sig.token_address.slice(0, 6).toUpperCase()
            const updatedAt = sig.updated_at ? relativeTime(sig.updated_at) : '最新'
            return (
              <div key={sig.id} className="opp-item" onClick={() => onOppClick(sym, score)}>
                <div className="opp-row">
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span className="opp-rank-badge">{RANK_MEDALS[idx] ?? `#${idx + 1}`}</span>
                    <span className="token-name">{sym}</span>
                  </div>
                  <span className="score-badge">{score} 分</span>
                </div>
                <div className="opp-detail">
                  <span>
                    <i className="fas fa-users" />
                    {sig.smart_wallet_count} 个聪明钱包买入
                  </span>
                  <span className="trend-up">
                    <i className="fas fa-clock" />
                    {updatedAt}
                  </span>
                </div>
                <div className="opp-progress-bar">
                  <div className="opp-progress-fill" style={{ width: `${Math.min(score, 100)}%` }} />
                </div>
              </div>
            )
          })
        ) : (
          <div className="empty-state" style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)' }}>
            暂无今日机会
          </div>
        )}
      </div>
    </div>
  )
}

// ── Section 3: Activity Card ─────────────────────
function ActivityCard({
  activities,
  loading,
  onActivityClick,
}: {
  activities: ActivityItem[]
  loading: boolean
  onActivityClick: (a: ActivityItem) => void
}) {
  return (
    <div className="card">
      <div className="card-header">
        <i className="fas fa-brain" />
        <h2>聪明钱动态</h2>
        <span className="card-header-actions" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Link to="/smart-wallets" className="btn btn-sm btn-ghost">全部 →</Link>
        </span>
      </div>
      <div className="activity-list">
        {loading ? (
          <div className="loading-state" style={{ padding: 32 }}>
            <div className="spinner" />
            <span>加载中...</span>
          </div>
        ) : activities.length === 0 ? (
          <div className="empty-state" style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)' }}>
            暂无聪明钱动态
          </div>
        ) : (
          activities.map((act) => {
            const shortAddr = `${act.walletAddress.slice(0, 6)}...${act.walletAddress.slice(-4)}`
            return (
              <div key={act.id} className="activity-item" onClick={() => onActivityClick(act)}>
                <div className="time-badge">{act.timeLabel}</div>
                <div className="activity-content">
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                    <span className="wallet-tag">#{act.walletRank} {shortAddr}</span>
                    <span className={act.action === 'buy' ? 'action-buy' : 'action-sell'}>
                      <i className={`fas ${act.action === 'buy' ? 'fa-cart-shopping' : 'fa-arrow-down'}`} />
                      {' '}{act.action === 'buy' ? '买入' : '卖出'} {act.token}
                    </span>
                  </div>
                  {act.amountUsd > 0 && (
                    <div className="activity-amount">
                      <i className="fas fa-dollar-sign" />
                      {formatUsd(act.amountUsd)}
                    </div>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}

// ── Section 4: Leaderboard Snapshot ──────────────
function LeaderboardSnapshot({
  wallets,
  loading,
}: {
  wallets: SmartWallet[]
  loading: boolean
}) {
  return (
    <div className="card leaderboard-snapshot">
      <div className="card-header">
        <i className="fas fa-medal" />
        <h2>聪明钱榜单</h2>
        <span className="card-header-actions">
          <Link to="/smart-wallets" className="btn btn-sm btn-ghost">全部 →</Link>
        </span>
      </div>

      <div className="snapshot-body">
      {loading ? (
        <div className="snapshot-grid">
          {[1, 2, 3].map((i) => (
            <div key={i} className="snapshot-card snapshot-skeleton" />
          ))}
        </div>
      ) : wallets.length === 0 ? (
        <div className="empty-state" style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)' }}>
          暂无聪明钱榜单数据
        </div>
      ) : (
        <div className="snapshot-grid">
          {wallets.map((w, idx) => {
            const score = parseFloat(w.score)
            const pnl = parseFloat(w.total_pnl)
            const winRate = parseFloat(w.win_rate)
            return (
              <Link
                key={w.id}
                to={`/smart-wallets/${w.wallet_address}`}
                className="snapshot-card"
              >
                <div className="snapshot-rank-row">
                  <span className="rank-medal">{RANK_MEDALS[idx]}</span>
                  <span className="snapshot-address">
                    {w.wallet_address.slice(0, 8)}...{w.wallet_address.slice(-4)}
                  </span>
                </div>
                <div className="snapshot-score-row">
                  <span className="snapshot-score-label">综合评分</span>
                  <span
                    className={`score-pill ${
                      score >= 90 ? 'score-high' : score >= 80 ? 'score-mid' : 'score-low'
                    }`}
                  >
                    {score.toFixed(1)}
                  </span>
                </div>
                <div className="progress-bar" style={{ marginBottom: 10 }}>
                  <div
                    className="progress-fill fill-success"
                    style={{ width: `${Math.min(score, 100)}%` }}
                  />
                </div>
                <div className="snapshot-metrics">
                  <div className="snapshot-metric">
                    <span className="snapshot-metric-label">累计PNL</span>
                    <span
                      className={`snapshot-metric-value ${pnl >= 0 ? 'text-success' : 'text-danger'}`}
                    >
                      {pnl >= 0 ? '+' : ''}{formatUsd(Math.abs(pnl))}
                    </span>
                  </div>
                  <div className="snapshot-metric">
                    <span className="snapshot-metric-label">胜率</span>
                    <span className="snapshot-metric-value">{winRate.toFixed(1)}%</span>
                  </div>
                  <div className="snapshot-metric">
                    <span className="snapshot-metric-label">总交易</span>
                    <span className="snapshot-metric-value">{w.total_trades}</span>
                  </div>
                </div>
              </Link>
            )
          })}
        </div>
      )}
      </div>
    </div>
  )
}

// ── Section 5a: Guest CTA ─────────────────────────
function GuestCTA() {
  return (
    <div className="guest-cta">
      <div className="guest-cta-icon">
        <i className="fas fa-rocket" />
      </div>
      <h3>连接钱包，一键复制聪明钱策略</h3>
      <p>追踪链上顶级交易者的每一笔操作，实时信号推送，自动跟单执行</p>
      <ConnectWallet className="connect-btn connect-btn-cta" />
    </div>
  )
}

// ── Toast ──────────────────────────────────────
function Toast({ msg }: { msg: string }) {
  return (
    <div
      style={{
        position: 'fixed',
        bottom: 30,
        left: '50%',
        transform: 'translateX(-50%)',
        background: '#1f2f40',
        color: '#fff',
        padding: '10px 24px',
        borderRadius: 40,
        fontSize: 13,
        fontWeight: 500,
        zIndex: 1000,
        whiteSpace: 'nowrap',
        boxShadow: '0 4px 16px rgba(0,0,0,.14)',
        animation: 'slideUp .2s ease',
        maxWidth: '90vw',
        textOverflow: 'ellipsis',
        overflow: 'hidden',
      }}
    >
      <i className="fas fa-bolt" style={{ marginRight: 7 }} />
      {msg}
    </div>
  )
}
