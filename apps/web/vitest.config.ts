import { fileURLToPath, URL } from 'node:url'

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

import { readAppPackageVersion, readGitCommitSha } from './app-version.config.ts'

const appVersion = readAppPackageVersion()
const appCommitSha = readGitCommitSha()

export default defineConfig({
  define: {
    __APP_COMMIT_SHA__: JSON.stringify(appCommitSha),
    __APP_COMMIT_SHORT_SHA__: JSON.stringify(appCommitSha.slice(0, 8)),
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    environmentOptions: {
      jsdom: {
        url: 'http://127.0.0.1:5173/',
      },
    },
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    restoreMocks: true,
    clearMocks: true,
  },
})
