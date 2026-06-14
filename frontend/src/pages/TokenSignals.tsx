import { useEffect, useState } from 'react'
import { apiClient } from '../api/client'
import { TokenSignal } from '../types'
import { Link } from 'react-router-dom'

export function TokenSignals() {
  const [signals, setSignals] = useState<TokenSignal[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchSignals()
  }, [])

  const fetchSignals = async () => {
    try {
      setLoading(true)
      const res = await apiClient.get('/token-signals?limit=20&min_consensus_score=50')
      setSignals(res.data.signals || [])
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
          <p className="mt-2 text-gray-600">Loading token signals...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Token Signals</h1>
          <p className="text-gray-600 mt-1">
            Tokens bought by smart wallets in the last 3 days
          </p>
        </div>
        <button
          onClick={fetchSignals}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
        >
          Refresh
        </button>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800">{error}</p>
        </div>
      )}

      {signals.length === 0 ? (
        <div className="bg-white rounded-lg shadow p-12 text-center">
          <p className="text-gray-500">No token signals found</p>
        </div>
      ) : (
        <div className="space-y-4">
          {signals.map((signal) => (
            <div
              key={signal.id}
              className="bg-white rounded-lg shadow hover:shadow-md transition p-6"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <h3 className="text-xl font-bold text-gray-900">
                      {signal.token_symbol || 'Unknown Token'}
                    </h3>
                    <span
                      className={`inline-flex px-3 py-1 text-sm font-semibold rounded-full ${
                        parseFloat(signal.consensus_score) >= 80
                          ? 'bg-green-100 text-green-800'
                          : parseFloat(signal.consensus_score) >= 60
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      Consensus: {parseFloat(signal.consensus_score).toFixed(1)}
                    </span>
                  </div>

                  <p className="text-sm text-gray-500 font-mono mb-4">
                    {signal.token_address}
                  </p>

                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <p className="text-xs text-gray-500 mb-1">
                        Smart Wallets
                      </p>
                      <p className="text-lg font-semibold text-gray-900">
                        {signal.smart_wallet_count}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 mb-1">
                        Total Buy Volume
                      </p>
                      <p className="text-lg font-semibold text-gray-900">
                        ${parseFloat(signal.total_buy_volume).toLocaleString()}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 mb-1">
                        Avg Buy Amount
                      </p>
                      <p className="text-lg font-semibold text-gray-900">
                        ${parseFloat(signal.avg_buy_amount).toLocaleString()}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 mb-1">
                        Time Period
                      </p>
                      <p className="text-sm text-gray-900">
                        {formatTime(signal.first_buy_time)} -
                        <br />
                        {formatTime(signal.last_buy_time)}
                      </p>
                    </div>
                  </div>

                  {signal.price_usd && (
                    <div className="mt-4 pt-4 border-t border-gray-200">
                      <div className="flex gap-6">
                        <div>
                          <p className="text-xs text-gray-500">Price (USD)</p>
                          <p className="text-sm font-medium text-gray-900">
                            ${parseFloat(signal.price_usd).toFixed(6)}
                          </p>
                        </div>
                        {signal.market_cap && (
                          <div>
                            <p className="text-xs text-gray-500">Market Cap</p>
                            <p className="text-sm font-medium text-gray-900">
                              ${parseFloat(signal.market_cap).toLocaleString()}
                            </p>
                          </div>
                        )}
                        {signal.liquidity_usd && (
                          <div>
                            <p className="text-xs text-gray-500">Liquidity</p>
                            <p className="text-sm font-medium text-gray-900">
                              ${parseFloat(signal.liquidity_usd).toLocaleString()}
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>

                <Link
                  to={`/token-signals/${signal.id}`}
                  className="ml-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition text-sm font-medium"
                >
                  View Details
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="text-sm font-medium text-gray-500 mb-1">
            Signal Period
          </h3>
          <p className="text-lg font-semibold text-gray-900">Last 3 Days</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="text-sm font-medium text-gray-500 mb-1">
            Total Signals
          </h3>
          <p className="text-lg font-semibold text-gray-900">
            {signals.length}
          </p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="text-sm font-medium text-gray-500 mb-1">
            Min Consensus
          </h3>
          <p className="text-lg font-semibold text-gray-900">50</p>
        </div>
      </div>
    </div>
  )
}
