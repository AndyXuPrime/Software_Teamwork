import { create } from 'zustand'

type TransitionState = {
  visible: boolean
  covering: boolean
  /** Slide overlay up to cover screen, then call onCovered */
  cover: (onCovered: () => void) => void
  /** Slide overlay down to reveal new page */
  reveal: () => void
}

export const usePageTransitionStore = create<TransitionState>((set) => ({
  visible: false,
  covering: false,
  cover: (onCovered) => {
    set({ visible: true, covering: false })
    // Force layout, then animate in
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        set({ covering: true })
        setTimeout(onCovered, 400)
      })
    })
  },
  reveal: () => {
    setTimeout(() => {
      set({ covering: false })
      setTimeout(() => set({ visible: false }), 400)
    }, 50)
  },
}))
