import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { apiClient } from '../api/client'
import { WalletTrade } from '../types'

export function WalletDetail() {
  const { address } = useParams<{ address: string }>()
  const [trades, setTrades] = useState<WalletTrade[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (address) {
      fetchTrades()
    }
  }, [address])

  const fetchTrades = async () => {
    try {
      setLoading(true)
      const res = await apiClient.get(`/wallet-history/${address}?limit=100`)
      setTrades(res.data.trades || [])
    } catch (err: any) {
      setError(err.response?.data?.error || err.message)
    } finally {
      setLoading(false)
    }
  }

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleString()
  }

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="text-center py-12">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"></div>
          <p className="mt-2 text-gray-600">Loading wallet history...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="mb-6">
        <Link
          to="/smart-wallets"
          className="text-blue-600 hover:text-blue-800 text-sm mb-2 inline-block"
        >
          ← Back to Smart Wallets
        </Link>
        <h1 className="text-3xl font-bold text-gray-900">Wallet Details</h1>
        <p className="text-gray-600 mt-1 font-mono">{address}</p>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800">{error}</p>
        </div>
      )}

      {trades.length === 0 ? (
        <div className="bg-white rounded-lg shadow p-12 text-center">
          <p className="text-gray-500">No trade history found</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Time
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    DEX
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Token In
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Token Out
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Amount (USD)
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    PNL
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Tx Hash
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {trades.map((trade) => (
                  <tr key={trade.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {formatTime(trade.block_time)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          trade.is_buy
                            ? 'bg-green-100 text-green-800'
                            : 'bg-red-100 text-red-800'
                        }`}
                      >
                        {trade.is_buy ? 'BUY' : 'SELL'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {trade.dex_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div>
                        <div className="font-medium">
                          {trade.token_in_symbol || 'Unknown'}
                        </div>
                        <div className="text-xs text-gray-500 font-mono">
                          {parseFloat(trade.amount_in).toFixed(4)}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div>
                        <div className="font-medium">
                          {trade.token_out_symbol || 'Unknown'}
                        </div>
                        <div className="text-xs text-gray-500 font-mono">
                          {parseFloat(trade.amount_out).toFixed(4)}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      ${parseFloat(trade.amount_usd).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {trade.pnl_usd && trade.pnl_percent ? (
                        <div className="text-sm">
                          <div
                            className={`font-medium ${
                              parseFloat(trade.pnl_usd) >= 0
                                ? 'text-green-600'
                                : 'text-red-600'
                            }`}
                          >
                            ${parseFloat(trade.pnl_usd).toLocaleString()}
                          </div>
                          <div
                            className={`text-xs ${
                              parseFloat(trade.pnl_percent) >= 0
                                ? 'text-green-600'
                                : 'text-red-600'
                            }`}
                          >
                            ({parseFloat(trade.pnl_percent).toFixed(2)}%)
                          </div>
                        </div>
                      ) : (
                        <span className="text-gray-400 text-sm">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <a
                        href={`https://etherscan.io/tx/${trade.tx_hash}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-blue-600 hover:text-blue-800 font-mono"
                      >
                        {trade.tx_hash.slice(0, 6)}...{trade.tx_hash.slice(-4)}
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="mt-6 bg-white rounded-lg shadow p-4">
        <h3 className="text-sm font-medium text-gray-500 mb-1">
          Total Trades Shown
        </h3>
        <p className="text-lg font-semibold text-gray-900">{trades.length}</p>
      </div>
    </div>
  )
}
