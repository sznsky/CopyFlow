/**
 * API 客户端封装
 * 统一处理 JWT 鉴权与错误响应
 */

const TOKEN_KEY = 'copyflow_token'

/** 从 localStorage 读取 JWT */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

/** 保存 JWT 到 localStorage */
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

/** 清除登录态 */
export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

/** 通用请求封装 */
async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(path, { ...options, headers })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `请求失败: ${res.status}`)
  }
  return data as T
}

// --- 认证 ---

export async function fetchNonce(address: string) {
  return request<{ nonce: string; message: string }>('/api/auth/nonce', {
    method: 'POST',
    body: JSON.stringify({ address }),
  })
}

export async function verifyLogin(address: string, message: string, signature: string) {
  return request<{ token: string }>('/api/auth/verify', {
    method: 'POST',
    body: JSON.stringify({ address, message, signature }),
  })
}

export async function fetchMe() {
  return request<{ id: number; wallet_address: string; email: string }>('/api/me')
}

export async function sendEmailCode(email: string, purpose = 'register') {
  return request<{ ok: boolean; message: string }>('/api/auth/email/send-code', {
    method: 'POST',
    body: JSON.stringify({ email, purpose }),
  })
}

export async function emailRegister(email: string, code: string, password: string) {
  return request<{ token: string }>('/api/auth/email/register', {
    method: 'POST',
    body: JSON.stringify({ email, code, password }),
  })
}

export async function emailLogin(email: string, password: string) {
  return request<{ token: string }>('/api/auth/email/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

// --- 元信息 ---

export async function fetchChains() {
  return request<import('../types').ChainMeta[]>('/api/meta/chains')
}

// --- 跟单配置 ---

export async function fetchConfigs() {
  return request<import('../types').CopyConfig[]>('/api/configs')
}

export async function createConfig(body: Record<string, unknown>) {
  return request<import('../types').CopyConfig>('/api/configs', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function updateConfig(id: number, body: Record<string, unknown>) {
  return request<import('../types').CopyConfig>(`/api/configs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export async function deleteConfig(id: number) {
  return request<{ ok: boolean }>(`/api/configs/${id}`, { method: 'DELETE' })
}

// --- 跟单钱包 ---

export async function fetchWallets() {
  return request<import('../types').CopyWallet[]>('/api/wallets')
}

export async function createWallet(chainId: number) {
  return request<{ id: number; chain_id: number; address: string; message: string }>(
    '/api/wallets',
    { method: 'POST', body: JSON.stringify({ chain_id: chainId }) },
  )
}

// --- 交易记录 ---

export async function fetchCopyTrades() {
  return request<import('../types').CopyTrade[]>('/api/copy-trades')
}

export async function fetchLeaderTrades() {
  return request<import('../types').LeaderTrade[]>('/api/leader-trades')
}

// --- 聪明钱 API ---

/** axios 风格错误对象，携带响应体，供页面读取 err.response.data.error */
type HttpError = Error & { response?: { data?: { error?: string } } }

/** 轻量 GET 封装，返回 { data }，自动附带 JWT */
async function httpGet<T>(path: string): Promise<{ data: T }> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(path, { headers })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(
      (data as { error?: string }).error || `请求失败: ${res.status}`,
    ) as HttpError
    err.response = { data }
    throw err
  }
  return { data: data as T }
}

/** axios 风格客户端，供 SmartWallets / TokenSignals / WalletDetail 使用 */
export const apiClient = {
  get: <T = unknown>(path: string) => httpGet<T>(path),
}
