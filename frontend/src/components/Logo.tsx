import { useId } from 'react'
import { Link } from 'react-router-dom'

interface LogoProps {
  size?: number
  showText?: boolean
  className?: string
}

/**
 * AlphFlow 品牌 Logo。
 * 书法风格上升曲线 + 渐变文字（AlphFlow）。
 * `size` 表示 Logo 的高度，宽度按 230:50 比例自适应。
 */
export function Logo({ size = 40, showText = true, className }: LogoProps) {
  const uid = useId().replace(/:/g, '')
  const gradText = `gradText-${uid}`
  const calli = `calli-${uid}`
  const shadow = `shadow-${uid}`

  const width = showText ? Math.round((size * 230) / 50) : Math.round((size * 48) / 50)

  return (
    <Link to="/" className={className ? `logo ${className}` : 'logo'} style={{ maxWidth: '100%' }}>
      <svg
        viewBox={showText ? '0 0 230 50' : '0 0 48 50'}
        style={{ width: `${width}px`, maxWidth: '100%', height: 'auto', display: 'block' }}
        aria-hidden="true"
      >
        <defs>
          <linearGradient id={gradText} x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#4A9EFF" />
            <stop offset="100%" stopColor="#6C5CE7" />
          </linearGradient>
          <linearGradient id={calli} x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#4A9EFF" />
            <stop offset="100%" stopColor="#6C5CE7" />
          </linearGradient>
          <filter id={shadow} x="-10%" y="-10%" width="120%" height="120%">
            <feDropShadow dx="1" dy="1" stdDeviation="1" floodColor="#000" floodOpacity="0.3" />
          </filter>
        </defs>

        {/* 图形：书法上升曲线 */}
        <path
          d="M16,26 C10,26 6,20 10,14 C14,8 24,8 28,14 C32,20 28,30 20,32 C16,34 14,30 14,28"
          stroke={`url(#${calli})`}
          strokeWidth="3.5"
          fill="none"
          strokeLinecap="round"
        />
        <path
          d="M14,28 C12,30 14,32 16,30"
          stroke={`url(#${calli})`}
          strokeWidth="2.5"
          fill="none"
          strokeLinecap="round"
        />
        <circle cx="16" cy="26" r="2" fill="#4A9EFF" />

        {showText && (
          <text
            x="40"
            y="37"
            fontFamily="'Brush Script MT', 'Apple Chancery', cursive"
            fontWeight="900"
            fontSize="35"
            fill={`url(#${gradText})`}
            fontStyle="italic"
            filter={`url(#${shadow})`}
            letterSpacing="0.5"
            stroke={`url(#${gradText})`}
            strokeWidth="0.8"
          >
            AlphFlow
          </text>
        )}
      </svg>
    </Link>
  )
}
