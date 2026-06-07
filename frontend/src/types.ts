/** 后端返回的链与 DEX 元信息 */
export interface ChainMeta {
  chain_id: number
  name: string
  native_symbol: string
  enabled: boolean
  dexes: { type: string; enabled: boolean }[]
}

/** 跟单配置 */
export interface CopyConfig {
  id: number
  user_id: number
  chain_id: number
  dex_type: string
  leader_address: string
  copy_mode: 'fixed' | 'ratio'
  copy_amount: string
  max_per_trade: string
  slippage_bps: number
  is_active: boolean
  created_at: string
  updated_at: string
}

/** 跟单专用钱包 */
export interface CopyWallet {
  id: number
  user_id: number
  chain_id: number
  address: string
  is_active: boolean
  created_at: string
}

/** 领头地址交易记录 */
export interface LeaderTrade {
  id: number
  chain_id: number
  leader_address: string
  tx_hash: string
  dex_type: string
  token_in: string
  token_out: string
  amount_in: string
  amount_out: string
  block_number: number
  detected_at: string
}

/** 用户跟单交易记录 */
export interface CopyTrade {
  id: number
  user_id: number
  config_id: number
  leader_trade_id: number
  tx_hash: string | null
  status: string
  amount_in: string
  token_out: string
  error_msg: string | null
  created_at: string
  leader_trade?: LeaderTrade
}

/** 当前登录用户 */
export interface User {
  id: number
  wallet_address: string
}
