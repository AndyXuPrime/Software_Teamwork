import { RouterProvider } from '@tanstack/react-router'

import { usePageTransitionStore } from '@/stores/page-transition-store'

import { AppProviders } from './providers'
import { router } from './router'

function TransitionOverlay() {
  const visible = usePageTransitionStore((s) => s.visible)
  const covering = usePageTransitionStore((s) => s.covering)

  if (!visible) return null

  return (
    <div
      aria-hidden
      className="fixed inset-0 z-[9999] flex flex-col items-center justify-center bg-white transition-transform duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)]"
      style={{ transform: covering ? 'translateY(0)' : 'translateY(100%)' }}
    >
      <svg className="mb-8 w-20 overflow-visible" viewBox="0 0 50 50">
        <circle
          r={25}
          cx={25}
          cy={25}
          className="animate-[circle_rotate_3s_ease-in_infinite]"
          fill="none"
          stroke="#1a1a2e"
          strokeWidth={12}
          strokeDasharray={160}
          strokeDashoffset={160}
          style={{ transformOrigin: 'center' }}
        />
      </svg>
      <p className="text-2xl font-black text-[#1a1a2e]">欢迎使用</p>
    </div>
  )
}

export function App() {
  return (
    <AppProviders>
      <RouterProvider router={router} />
      <TransitionOverlay />
    </AppProviders>
  )
}
