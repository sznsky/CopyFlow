import { useEffect, useState } from 'react'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'
import type { ChainMeta, CopyWallet } from '../types'

/** 跟单钱包管理：每个链一个专用钱包，需充值 BNB/ETH */
export function Wallets() {
  const { user } = useAuth()
  const [chains, setChains] = useState<ChainMeta[]>([])
  const [wallets, setWallets] = useState<CopyWallet[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState<number | null>(null)
  const [message, setMessage] = useState('')

  const load = async () => {
    const [c, w] = await Promise.all([api.fetchChains(), api.fetchWallets()])
    setChains(c.filter((ch) => ch.enabled))
    setWallets(w)
  }

  useEffect(() => {
    if (!user) return
    load().finally(() => setLoading(false))
  }, [user])

  const handleCreate = async (chainId: number) => {
    setCreating(chainId)
    setMessage('')
    try {
      const res = await api.createWallet(chainId)
      setMessage(res.message)
      await load()
    } catch (e) {
      setMessage(e instanceof Error ? e.message : '创建失败')
    } finally {
      setCreating(null)
    }
  }

  if (!user) return <p className="text-muted">请先登录</p>
  if (loading) return <p className="text-muted">加载中...</p>

  return (
    <div className="page">
      <h2>跟单钱包</h2>
      <p className="text-muted">
        跟单钱包由系统生成，私钥加密存储在服务端。请向钱包地址充值原生币（BNB/ETH）用于自动跟单。
      </p>

      {message && <div className="info-banner">{message}</div>}

      <div className="wallet-grid">
        {chains.map((ch) => {
          const wallet = wallets.find((w) => w.chain_id === ch.chain_id)
          return (
            <div key={ch.chain_id} className="card wallet-card">
              <h3>{ch.name}</h3>
              <p className="text-muted">原生币: {ch.native_symbol}</p>
              {wallet ? (
                <>
                  <p className="mono wallet-address">{wallet.address}</p>
                  <span className="badge badge-success">已创建</span>
                </>
              ) : (
                <button
                  className="btn btn-primary"
                  disabled={creating === ch.chain_id}
                  onClick={() => handleCreate(ch.chain_id)}
                >
                  {creating === ch.chain_id ? '生成中...' : '生成跟单钱包'}
                </button>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
