import { useEffect, useRef, useState } from 'react'
import { useAccount, useConnect, useSignMessage } from 'wagmi'
import { injected } from 'wagmi/connectors'
import * as api from '../api/client'
import { useAuth } from '../context/AuthContext'

type Mode = 'login' | 'register'
type RegisterStep = 'form' | 'verify'

interface AuthModalProps {
  mode: Mode
  onClose: () => void
  onSwitch: (mode: Mode) => void
}

/** 登录 / 注册弹窗（邮箱验证码 + MetaMask） */
export function AuthModal({ mode, onClose, onSwitch }: AuthModalProps) {
  const { loginWithToken } = useAuth()
  const { connectAsync } = useConnect()
  const { address } = useAccount()
  const { signMessageAsync } = useSignMessage()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [code, setCode] = useState(['', '', '', '', '', ''])
  const [registerStep, setRegisterStep] = useState<RegisterStep>('form')
  const [countdown, setCountdown] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const inputsRef = useRef<(HTMLInputElement | null)[]>([])

  useEffect(() => {
    if (countdown <= 0) return
    const t = setTimeout(() => setCountdown(countdown - 1), 1000)
    return () => clearTimeout(t)
  }, [countdown])

  const handleSendCode = async () => {
    setError('')
    if (!email) {
      setError('请输入邮箱')
      return
    }
    if (password.length < 6) {
      setError('密码至少 6 位')
      return
    }
    if (password !== confirmPwd) {
      setError('两次密码不一致')
      return
    }
    setLoading(true)
    try {
      await api.sendEmailCode(email, 'register')
      setRegisterStep('verify')
      setCountdown(60)
      setTimeout(() => inputsRef.current[0]?.focus(), 100)
    } catch (e) {
      setError(e instanceof Error ? e.message : '发送失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCodeChange = (idx: number, val: string) => {
    if (!/^\d?$/.test(val)) return
    const next = [...code]
    next[idx] = val
    setCode(next)
    if (val && idx < 5) inputsRef.current[idx + 1]?.focus()
  }

  const handleCodeKeyDown = (idx: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !code[idx] && idx > 0) {
      inputsRef.current[idx - 1]?.focus()
    }
  }

  const handleRegister = async () => {
    const fullCode = code.join('')
    if (fullCode.length !== 6) {
      setError('请输入 6 位验证码')
      return
    }
    setLoading(true)
    setError('')
    try {
      const { token } = await api.emailRegister(email, fullCode, password)
      await loginWithToken(token)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  const handleEmailLogin = async () => {
    setLoading(true)
    setError('')
    try {
      const { token } = await api.emailLogin(email, password)
      await loginWithToken(token)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleMetaMask = async () => {
    setLoading(true)
    setError('')
    try {
      let addr = address
      if (!addr) {
        const result = await connectAsync({ connector: injected() })
        addr = result.accounts[0]
      }
      const { message } = await api.fetchNonce(addr)
      const signature = await signMessageAsync({ message })
      const { token } = await api.verifyLogin(addr, message, signature)
      await loginWithToken(token)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'MetaMask 登录失败')
    } finally {
      setLoading(false)
    }
  }

  const title = mode === 'register' ? '注册' : '登录'

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          {registerStep === 'verify' && mode === 'register' ? (
            <button className="modal-back" onClick={() => setRegisterStep('form')}>←</button>
          ) : (
            <span />
          )}
          <h2>{registerStep === 'verify' ? '邮箱验证码' : title}</h2>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>

        {mode === 'register' && registerStep === 'form' && (
          <>
            <p className="modal-sub">
              已有账号？<button className="link-btn" onClick={() => onSwitch('login')}>去登录</button>
            </p>
            <label className="modal-field">
              邮箱
              <input
                type="email"
                placeholder="输入邮箱"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
            <label className="modal-field">
              密码
              <input
                type="password"
                placeholder="至少 6 位"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
            <label className="modal-field">
              确认密码
              <input
                type="password"
                placeholder="再次输入密码"
                value={confirmPwd}
                onChange={(e) => setConfirmPwd(e.target.value)}
              />
            </label>
            {error && <p className="error-text">{error}</p>}
            <button className="btn btn-primary btn-block" onClick={handleSendCode} disabled={loading}>
              {loading ? '发送中...' : '发送验证码'}
            </button>
            <div className="modal-divider"><span>其它注册方式</span></div>
            <button className="wallet-option" onClick={handleMetaMask} disabled={loading}>
              <img src="https://upload.wikimedia.org/wikipedia/commons/3/36/MetaMask_Fox.svg" alt="MetaMask" className="wallet-icon" />
              <span>MetaMask</span>
            </button>
          </>
        )}

        {mode === 'register' && registerStep === 'verify' && (
          <>
            <p className="modal-sub">
              请输入发送至邮箱 <strong>{email}</strong> 的 6 位验证码
            </p>
            <div className="code-inputs">
              {code.map((d, i) => (
                <input
                  key={i}
                  ref={(el) => { inputsRef.current[i] = el }}
                  type="text"
                  inputMode="numeric"
                  maxLength={1}
                  value={d}
                  onChange={(e) => handleCodeChange(i, e.target.value)}
                  onKeyDown={(e) => handleCodeKeyDown(i, e)}
                />
              ))}
            </div>
            <div className="code-actions">
              {countdown > 0 ? (
                <span className="text-muted">重新获取 ({countdown}秒)</span>
              ) : (
                <button className="link-btn" onClick={handleSendCode}>重新获取</button>
              )}
            </div>
            {error && <p className="error-text">{error}</p>}
            <button className="btn btn-primary btn-block" onClick={handleRegister} disabled={loading}>
              {loading ? '注册中...' : '注册'}
            </button>
            <p className="modal-hint">验证码 5 分钟内有效</p>
          </>
        )}

        {mode === 'login' && (
          <>
            <p className="modal-sub">
              没有账号？<button className="link-btn" onClick={() => onSwitch('register')}>去注册</button>
            </p>
            <label className="modal-field">
              邮箱
              <input
                type="email"
                placeholder="输入邮箱"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
            <label className="modal-field">
              密码
              <input
                type="password"
                placeholder="输入密码"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
            {error && <p className="error-text">{error}</p>}
            <button className="btn btn-primary btn-block" onClick={handleEmailLogin} disabled={loading}>
              {loading ? '登录中...' : '登录'}
            </button>
            <div className="modal-divider"><span>其它登录方式</span></div>
            <button className="wallet-option" onClick={handleMetaMask} disabled={loading}>
              <img src="https://upload.wikimedia.org/wikipedia/commons/3/36/MetaMask_Fox.svg" alt="MetaMask" className="wallet-icon" />
              <span>MetaMask</span>
            </button>
          </>
        )}
      </div>
    </div>
  )
}
