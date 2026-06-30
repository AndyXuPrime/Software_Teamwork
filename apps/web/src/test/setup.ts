import '@testing-library/jest-dom/vitest'

import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach, vi } from 'vitest'

import { resetApiClientForTests } from '@/api/client'
import { useAuthStore } from '@/stores/auth-store'

beforeEach(() => {
  resetApiClientForTests()
  vi.stubEnv('VITE_API_BASE_URL', 'http://127.0.0.1/api/v1')
})

afterEach(() => {
  cleanup()
  resetApiClientForTests()
  vi.unstubAllEnvs()
  vi.unstubAllGlobals()
  localStorage.clear()
  sessionStorage.clear()
  useAuthStore.setState({
    accessToken: null,
    error: null,
    status: 'anonymous',
    user: null,
    userName: null,
  })
})
