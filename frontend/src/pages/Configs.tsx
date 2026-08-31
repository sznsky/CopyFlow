import { FormEvent, useEffect, useState } from 'react'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'
import type { ChainMeta, CopyConfig } from '../types'

/** 跟单配置管理页 */
export function Configs() {
  const { user } = useAuth()
  const [chains, setChains] = useState<ChainMeta[]>([])
  const [configs, setConfigs] = useState<CopyConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)

  const load = async () => {
    const [c, cfg] = await Promise.all([api.fetchChains(), api.fetchConfigs()])
    setChains(c.filter((ch) => ch.enabled))
    setConfigs(cfg)
  }

  useEffect(() => {
    if (!user) return
    load().finally(() => setLoading(false))
  }, [user])

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除该配置？')) return
    await api.deleteConfig(id)
    await load()
  }

  const handleToggle = async (cfg: CopyConfig) => {
    await api.updateConfig(cfg.id, { is_active: !cfg.is_active })
    await load()
  }

  if (!user) return <p className="text-muted">请先登录</p>
  if (loading) return <p className="text-muted">加载中...</p>

  return (
    <div className="page-wrap">
      <div className="page-header">
        <div>
          <h2>跟单配置</h2>
          <p className="page-desc">设置要跟单的地址、链和交易策略</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowForm(!showForm)}>
          {showForm ? '取消' : '+ 新增配置'}
        </button>
      </div>

      {showForm && (
        <ConfigForm
          chains={chains}
          onSuccess={async () => {
            setShowForm(false)
            await load()
          }}
          onError={setError}
        />
      )}

      {error && <p className="error-text">{error}</p>}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>链</th>
              <th>DEX</th>
              <th>领头地址</th>
              <th>模式</th>
              <th>金额/比例</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {configs.length === 0 ? (
              <tr>
                <td colSpan={7} className="text-muted" style={{ textAlign: 'center', padding: '32px 14px' }}>
                  暂无配置，点击「新增配置」开始
                </td>
              </tr>
            ) : (
              configs.map((cfg) => (
                <tr key={cfg.id}>
                  <td>{cfg.chain_id}</td>
                  <td style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{cfg.dex_type}</td>
                  <td className="mono">{cfg.leader_address}</td>
                  <td>
                    <span className="badge badge-pending">
                      {cfg.copy_mode === 'ratio' ? '等比例' : '固定金额'}
                    </span>
                  </td>
                  <td className="mono">{cfg.copy_amount}</td>
                  <td>
                    <span className={`badge ${cfg.is_active ? 'badge-success' : 'badge-skipped'}`}>
                      {cfg.is_active ? '启用' : '停用'}
                    </span>
                  </td>
                  <td className="actions">
                    <button className="btn btn-sm btn-secondary" onClick={() => handleToggle(cfg)}>
                      {cfg.is_active ? '停用' : '启用'}
                    </button>
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(cfg.id)}>
                      删除
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

/** 新增跟单配置表单 */
function ConfigForm({
  chains,
  onSuccess,
  onError,
}: {
  chains: ChainMeta[]
  onSuccess: () => void
  onError: (msg: string) => void
}) {
  const firstChain = chains[0]
  const firstDex = firstChain?.dexes.find((d) => d.enabled)

  const [chainId, setChainId] = useState(firstChain?.chain_id ?? 56)
  const [dexType, setDexType] = useState(firstDex?.type ?? 'pancake_v2')
  const [leaderAddress, setLeaderAddress] = useState('')
  const [copyMode, setCopyMode] = useState<'ratio' | 'fixed'>('ratio')
  const [copyAmount, setCopyAmount] = useState('0.1')
  const [maxPerTrade, setMaxPerTrade] = useState('1')
  const [slippageBps, setSlippageBps] = useState(300)
  const [submitting, setSubmitting] = useState(false)

  const selectedChain = chains.find((c) => c.chain_id === chainId)
  const dexOptions = selectedChain?.dexes.filter((d) => d.enabled) ?? []

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    onError('')
    try {
      await api.createConfig({
        chain_id: chainId,
        dex_type: dexType,
        leader_address: leaderAddress,
        copy_mode: copyMode,
        copy_amount: parseFloat(copyAmount),
        max_per_trade: parseFloat(maxPerTrade),
        slippage_bps: slippageBps,
        is_active: true,
      })
      onSuccess()
    } catch (err) {
      onError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form className="card form-card" onSubmit={handleSubmit}>
      <h3>新增跟单配置</h3>
      <div className="form-grid">
        <label>
          链
          <select
            value={chainId}
            onChange={(e) => {
              const id = Number(e.target.value)
              setChainId(id)
              const ch = chains.find((c) => c.chain_id === id)
              const dex = ch?.dexes.find((d) => d.enabled)
              if (dex) setDexType(dex.type)
            }}
          >
            {chains.map((ch) => (
              <option key={ch.chain_id} value={ch.chain_id}>{ch.name}</option>
            ))}
          </select>
        </label>
        <label>
          DEX
          <select value={dexType} onChange={(e) => setDexType(e.target.value)}>
            {dexOptions.map((d) => (
              <option key={d.type} value={d.type}>{d.type}</option>
            ))}
          </select>
        </label>
        <label className="full-width">
          领头地址
          <input
            value={leaderAddress}
            onChange={(e) => setLeaderAddress(e.target.value)}
            placeholder="0x..."
            required
          />
        </label>
        <label>
          跟单模式
          <select value={copyMode} onChange={(e) => setCopyMode(e.target.value as 'ratio' | 'fixed')}>
            <option value="ratio">等比例（如 0.1 = 10%）</option>
            <option value="fixed">固定金额（BNB/ETH）</option>
          </select>
        </label>
        <label>
          {copyMode === 'ratio' ? '跟单比例' : '固定金额'}
          <input
            type="number"
            step="0.01"
            min="0"
            value={copyAmount}
            onChange={(e) => setCopyAmount(e.target.value)}
            required
          />
        </label>
        <label>
          单笔上限（BNB/ETH）
          <input
            type="number"
            step="0.01"
            min="0"
            value={maxPerTrade}
            onChange={(e) => setMaxPerTrade(e.target.value)}
          />
        </label>
        <label>
          滑点（bps，300 = 3%）
          <input
            type="number"
            value={slippageBps}
            onChange={(e) => setSlippageBps(Number(e.target.value))}
          />
        </label>
      </div>
      <button className="btn btn-primary" type="submit" disabled={submitting}>
        {submitting ? '保存中...' : '保存配置'}
      </button>
    </form>
  )
}
