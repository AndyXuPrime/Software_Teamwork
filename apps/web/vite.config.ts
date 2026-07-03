import { fileURLToPath, URL } from 'node:url'

import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

import { readAppPackageVersion, readGitCommitSha } from './app-version.config.ts'

const appVersion = readAppPackageVersion()
const appCommitSha = readGitCommitSha()

// https://vite.dev/config/
export default defineConfig({
  define: {
    __APP_COMMIT_SHA__: JSON.stringify(appCommitSha),
    __APP_COMMIT_SHORT_SHA__: JSON.stringify(appCommitSha.slice(0, 8)),
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: false,
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['./src/test/setup.ts'],
  },
})
