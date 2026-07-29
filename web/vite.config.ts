import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 4173,
    host: '127.0.0.1',
  },
  build: {
    manifest: true,
    rollupOptions: {
      input: {
        operations: fileURLToPath(new URL('./index.html', import.meta.url)),
        pilot: fileURLToPath(new URL('./pilot.html', import.meta.url)),
      },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    setupFiles: ['./src/test-setup.ts'],
  },
})
