/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load environment variables
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [
      vue(),
      vueDevTools(),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    // Use hash for router compatibility
    build: {
      outDir: 'dist',
      sourcemap: env.VITE_SOURCEMAP === 'true',
    },
    test: {
      environment: 'jsdom',
      globals: true,
      include: ['src/**/*.{test,spec}.{js,ts}'],
      setupFiles: ['src/__tests__/setup.ts'],
    },
    // Server configuration
    server: {
      port: parseInt(env.VITE_PORT || '8080'),
      host: true,
      // Enable HMR on all interfaces
      hmr: {
        host: '0.0.0.0',
        port: parseInt(env.VITE_PORT || '8080'),
      },
    },
    // Preview configuration
    preview: {
      port: parseInt(env.VITE_PORT || '8080'),
      host: true,
    },
    // Define global variables
    define: {
      __APP_ENV__: JSON.stringify(env.VITE_APP_ENV || 'development'),
    },
  }
})
