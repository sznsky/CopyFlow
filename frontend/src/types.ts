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
  email: string
}

/** 聪明钱包 */
export interface SmartWallet {
  id: number
  wallet_address: string
  chain_id: number
  score: string
  total_pnl: string
  win_rate: string
  profit_loss_ratio: string
  max_drawdown: string
  mainstream_ratio: string
  trade_frequency: string
  total_trades: number
  winning_trades: number
  total_volume: string
  evaluation_start_date: string
  evaluation_end_date: string
  rank_position: number | null
  is_top_wallet: boolean
  created_at: string
  updated_at: string
}

/** 代币信号 */
export interface TokenSignal {
  id: number
  token_address: string
  token_symbol: string | null
  token_name: string | null
  chain_id: number
  smart_wallet_count: number
  total_buy_volume: string
  avg_buy_amount: string
  first_buy_time: string
  last_buy_time: string
  consensus_score: string
  price_usd: string | null
  market_cap: string | null
  liquidity_usd: string | null
  signal_start_date: string
  signal_end_date: string
  created_at: string
  updated_at: string
}

/** 代币信号详情 */
export interface TokenSignalDetail {
  id: number
  signal_id: number
  wallet_address: string
  wallet_score: string
  trade_id: number
  buy_amount_usd: string
  buy_time: string
  created_at: string
}

/** 钱包交易历史 */
export interface WalletTrade {
  id: number
  wallet_address: string
  chain_id: number
  tx_hash: string
  block_number: number
  block_time: string
  dex_name: string
  pool_address: string
  token_in: string
  token_out: string
  token_in_symbol: string | null
  token_out_symbol: string | null
  amount_in: string
  amount_out: string
  amount_usd: string
  is_buy: boolean
  pnl_usd: string | null
  pnl_percent: string | null
  holding_duration_hours: number | null
  created_at: string
}
