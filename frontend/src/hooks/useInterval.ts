import { useEffect, useRef } from 'react'

/** 定时执行回调，组件卸载时自动清理 */
export function useInterval(callback: () => void, delayMs: number | null) {
  const saved = useRef(callback)
  saved.current = callback

  useEffect(() => {
    if (delayMs === null) return
    const id = setInterval(() => saved.current(), delayMs)
    return () => clearInterval(id)
  }, [delayMs])
}
