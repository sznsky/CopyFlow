import { Link } from 'react-router-dom'

/** CopyFlow 品牌 Logo + 名称 */
export function Logo() {
  return (
    <Link to="/" className="logo-brand">
      <img src="/logo.png" alt="CopyFlow" className="logo-icon" />
      <span className="logo-text">CopyFlow</span>
    </Link>
  )
}
