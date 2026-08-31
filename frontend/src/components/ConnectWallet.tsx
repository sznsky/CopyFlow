import { useEffect, useRef, useState } from 'react'
import { useAccount, useConnect, useDisconnect, useSignMessage } from 'wagmi'
import { injected } from 'wagmi/connectors'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'

type Status = 'idle' | 'connecting' | 'signing' | 'error'

/**
 * 连接钱包按钮（完整流程）
 *
 * 未连接 → 点击连接 MetaMask
 * 已连接 → 自动弹出签名 → 登录
 * 已登录 → 由 Layout 负责展示用户标识
 */
export function ConnectWallet({ className }: { className?: string }) {
  const { address, isConnected } = useAccount()
  const { connect, isPending: isConnecting } = useConnect()
  const { disconnect } = useDisconnect()
  const { signMessageAsync } = useSignMessage()
  const { login, user, loading: authLoading } = useAuth()

  const [status, setStatus] = useState<Status>('idle')
  const [errMsg, setErrMsg] = useState('')
  const signingRef = useRef(false)   // 防止 useEffect 重复触发签名

  // 钱包连接成功后自动签名登录。
  // authLoading 期间（登录态恢复中）不触发，避免刷新页面时误弹签名窗。
  useEffect(() => {
    if (authLoading) return
    if (!isConnected || !address || user || signingRef.current) return
    signingRef.current = true
    handleSign(address).finally(() => { signingRef.current = false })
  }, [isConnected, address, user, authLoading])

  const handleConnect = () => {
    setErrMsg('')
    setStatus('connecting')
    connect(
      { connector: injected() },
      {
        onError: (e) => {
          setStatus('error')
          setErrMsg(e.message.includes('rejected') ? '用户取消了连接' : e.message)
        },
      },
    )
  }

  const handleSign = async (addr: string) => {
    setStatus('signing')
    setErrMsg('')
    try {
      const { message } = await api.fetchNonce(addr)
      const signature = await signMessageAsync({ message })
      await login(addr, message, signature)
      setStatus('idle')
    } catch (e) {
      const msg = e instanceof Error ? e.message : '签名失败'
      // 用户手动拒签
      if (msg.toLowerCase().includes('reject') || msg.toLowerCase().includes('denied') || msg.toLowerCase().includes('cancel')) {
        setErrMsg('已取消签名，请重新签名以完成登录')
      } else {
        setErrMsg(msg)
      }
      setStatus('error')
    }
  }

  const handleRetrySign = () => {
    if (!address) return
    signingRef.current = false
    handleSign(address)
  }

  const handleDisconnect = () => {
    disconnect()
    api.clearToken()
    setStatus('idle')
    setErrMsg('')
    signingRef.current = false
  }

  // ── 已连接但尚未完成签名 (signing / error) ──
  if (isConnected && !user) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
        {status === 'signing' || isConnecting ? (
          <button className={className ?? 'connect-btn'} disabled>
            <i className="fas fa-circle-notch fa-spin" />
            签名中...
          </button>
        ) : (
          <button className={className ?? 'connect-btn'} onClick={handleRetrySign}>
            <i className="fas fa-pen-to-square" />
            点击签名登录
          </button>
        )}
        <button
          onClick={handleDisconnect}
          style={{
            background: 'none', border: 'none', fontSize: 12,
            color: 'var(--text-tip)', cursor: 'pointer', padding: '4px 8px',
          }}
        >
          断开
        </button>
        {errMsg && (
          <span style={{ fontSize: 12, color: 'var(--danger)', width: '100%' }}>
            <i className="fas fa-circle-exclamation" style={{ marginRight: 4 }} />
            {errMsg}
          </span>
        )}
      </div>
    )
  }

  // ── 未连接 ──
  return (
    <button
      className={className ?? 'connect-btn'}
      onClick={handleConnect}
      disabled={status === 'connecting' || isConnecting}
    >
      {status === 'connecting' || isConnecting ? (
        <>
          <i className="fas fa-circle-notch fa-spin" />
          连接中...
        </>
      ) : (
        <>
          <i className="fas fa-plug" />
          连接钱包
        </>
      )}
    </button>
  )
}
