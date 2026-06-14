import { useState } from 'react'
import { AuthModal } from './AuthModal'

/** 右上角注册 / 登录按钮 */
export function AuthButtons() {
  const [modal, setModal] = useState<'login' | 'register' | null>(null)

  return (
    <>
      <button className="btn btn-ghost" onClick={() => setModal('login')}>登录</button>
      <button className="btn btn-primary" onClick={() => setModal('register')}>注册</button>
      {modal && (
        <AuthModal
          mode={modal}
          onClose={() => setModal(null)}
          onSwitch={(m) => setModal(m)}
        />
      )}
    </>
  )
}
