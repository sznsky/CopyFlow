import { useState } from 'react'
import { useAccount, useConnect, useDisconnect, useSignMessage } from 'wagmi'
import { injected } from 'wagmi/connectors'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'

/** 钱包连接 + SIWE 签名登录 */
export function ConnectWallet() {
  const { address, isConnected } = useAccount()
  const { connect } = useConnect()
  const { disconnect } = useDisconnect()
  const { signMessageAsync } = useSignMessage()
  const { login } = useAuth()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleConnect = () => {
    connect({ connector: injected() })
  }

  const handleSignIn = async () => {
    if (!address) return
    setLoading(true)
    setError('')
    try {
      // 1. 向后端获取 nonce 和待签名消息
      const { message } = await api.fetchNonce(address)
      // 2. 钱包签名
      const signature = await signMessageAsync({ message })
      // 3. 后端验证签名，返回 JWT
      await login(address, message, signature)
    } catch (e) {
      setError(e instanceof Error ? e.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  if (!isConnected) {
    return (
      <button className="btn btn-primary" onClick={handleConnect}>
        连接钱包
      </button>
    )
  }

  return (
    <div className="connect-wallet">
      {!loading && (
        <button className="btn btn-primary" onClick={handleSignIn}>
          签名登录
        </button>
      )}
      {loading && <span className="text-muted">签名中...</span>}
      <button className="btn btn-ghost" onClick={() => disconnect()}>
        断开
      </button>
      {error && <span className="error-text">{error}</span>}
    </div>
  )
}
