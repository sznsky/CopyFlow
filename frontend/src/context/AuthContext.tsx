import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import * as api from '../api/client'
import type { User } from '../types'

interface AuthContextValue {
  user: User | null
  loading: boolean
  login: (address: string, message: string, signature: string) => Promise<void>
  logout: () => void
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

/** 全局登录态管理：JWT + 用户信息 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    if (!api.getToken()) {
      setUser(null)
      setLoading(false)
      return
    }
    try {
      const me = await api.fetchMe()
      setUser({ id: me.id, wallet_address: me.wallet_address })
    } catch {
      api.clearToken()
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  const login = async (address: string, message: string, signature: string) => {
    const { token } = await api.verifyLogin(address, message, signature)
    api.setToken(token)
    await refresh()
  }

  const logout = () => {
    api.clearToken()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth 必须在 AuthProvider 内使用')
  return ctx
}
