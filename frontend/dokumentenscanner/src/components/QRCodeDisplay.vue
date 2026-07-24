<script setup lang="ts">
import { computed } from 'vue'
import { useSessionStore } from '../stores/sessionStore'

const sessionStore = useSessionStore()

const qrCodeUrl = computed(() => {
  if (!sessionStore.sessionID) return ''
  const backendURL = sessionStore.backendURL
  return `${backendURL}/api/session/${sessionStore.sessionID}/qrcode`
})
</script>

<template>
  <div class="qr-display-container">
    <div class="qr-header">
      <h2>Scan to Connect</h2>
      <p class="info">Open your phone camera and scan this QR code</p>
    </div>

    <div class="qr-code-wrapper">
      <img
        v-if="qrCodeUrl"
        :src="qrCodeUrl"
        alt="QR Code"
        class="qr-code-image"
      />
      <div v-else class="qr-placeholder">
        <div class="spinner"></div>
        <p>Generating QR Code...</p>
      </div>
    </div>

    <div class="pin-display">
      <p class="pin-label">Or enter PIN manually:</p>
      <p class="pin-value">{{ sessionStore.pin }}</p>
    </div>

    <div class="session-info">
      <div class="waiting-badge">
        <span class="pulse-dot"></span>
        <span>Waiting for mobile upload...</span>
      </div>
      <p class="session-id">{{ sessionStore.sessionID }}</p>
    </div>
  </div>
</template>

<style scoped>
.qr-display-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  max-width: 440px;
  width: 100%;
  padding: 40px 32px;
  background: var(--color-surface);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
}

.qr-header {
  text-align: center;
}

.qr-header h2 {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-text);
  margin-bottom: 6px;
}

.info {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.qr-code-wrapper {
  width: 260px;
  height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 3px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  background: white;
  box-shadow: var(--shadow-md);
  transition: var(--transition);
}

.qr-code-wrapper:hover {
  box-shadow: var(--shadow-lg);
}

.qr-code-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.qr-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--color-text-muted);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.pin-display {
  text-align: center;
  padding: 18px 24px;
  background: linear-gradient(135deg, #eef2ff, #faf5ff);
  border: 1px solid var(--color-primary-light);
  border-radius: var(--radius-md);
  width: 100%;
}

.pin-label {
  color: var(--color-text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pin-value {
  font-size: 2.4rem;
  font-weight: 800;
  letter-spacing: 0.35em;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  font-family: 'SF Mono', 'Fira Code', 'Courier New', monospace;
}

.session-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.waiting-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: rgba(99, 102, 241, 0.08);
  border-radius: 20px;
  color: var(--color-primary);
  font-size: 0.85rem;
  font-weight: 500;
}

.pulse-dot {
  width: 8px;
  height: 8px;
  background: var(--color-primary);
  border-radius: 50%;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(1.4); }
}

.session-id {
  color: var(--color-text-muted);
  font-size: 0.7rem;
  word-break: break-all;
  text-align: center;
  font-family: 'SF Mono', 'Fira Code', monospace;
}
</style>
