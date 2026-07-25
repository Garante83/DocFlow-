<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useSessionStore } from '../stores/sessionStore'
import { apiService } from '../utils/api'
import { websocketClient } from '../utils/websocket'
import QRCodeDisplay from '../components/QRCodeDisplay.vue'

const sessionStore = useSessionStore()

type ViewState = 'loading' | 'error' | 'qr_display' | 'confirm_download' | 'waiting_confirm'
const currentView = ref<ViewState>('loading')
const errorMessage = ref<string | null>(null)

onMounted(async () => {
  await initializeSession()
})

onUnmounted(() => {
  websocketClient.off('image_uploaded', onImageUploaded)
  websocketClient.off('download_confirmed', onDownloadConfirmed)
  websocketClient.disconnect()
})

async function initializeSession() {
  currentView.value = 'loading'
  errorMessage.value = null

  websocketClient.off('image_uploaded', onImageUploaded)
  websocketClient.off('download_confirmed', onDownloadConfirmed)

  try {
    const response = await apiService.createSession()
    sessionStore.setSessionID(response.session_id)
    sessionStore.setPIN(response.pin)
    sessionStore.setStatus('waiting_for_pin')

    websocketClient.connect(response.session_id)

    websocketClient.on('image_uploaded', onImageUploaded)
    websocketClient.on('download_confirmed', onDownloadConfirmed)

    currentView.value = 'qr_display'
  } catch (error) {
    console.error('Error creating session:', error)
    errorMessage.value = 'Failed to create session. Please check your connection.'
    currentView.value = 'error'
  }
}

function onImageUploaded() {
  currentView.value = 'confirm_download'
}

function onDownloadConfirmed() {
  currentView.value = 'confirm_download'
  downloadPDF()
}

function requestDownload() {
  currentView.value = 'waiting_confirm'
  websocketClient.send('download_request', { session_id: sessionStore.sessionID })
}

async function downloadPDF() {
  if (!sessionStore.sessionID) return
  try {
    const blob = await apiService.downloadPDF(sessionStore.sessionID)
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'document.pdf'
    document.body.appendChild(a)
    a.click()
    window.URL.revokeObjectURL(url)
    document.body.removeChild(a)
  } catch (error) {
    console.error('Download failed:', error)
    errorMessage.value = 'Download failed. Please try again.'
    currentView.value = 'confirm_download'
  }
}
</script>

<template>
  <div class="desktop-view">
    <div class="header">
      <div class="logo">
        <span class="logo-icon">&#128196;</span>
      </div>
      <h1>Document Scanner</h1>
      <p class="subtitle">Scan and convert your documents to PDF</p>
    </div>

    <div class="main-content">
      <!-- Loading state -->
      <div v-if="currentView === 'loading'" class="card">
        <div class="spinner"></div>
        <p class="loading-text">Initializing session...</p>
      </div>

      <!-- Error state -->
      <div v-else-if="currentView === 'error'" class="card card-error">
        <div class="error-icon">&#9888;</div>
        <p class="error-message">{{ errorMessage }}</p>
        <button @click="initializeSession" class="btn btn-primary">Retry</button>
      </div>

      <!-- QR Code + PIN display -->
      <QRCodeDisplay v-else-if="currentView === 'qr_display'" />

      <!-- Image received + Download -->
      <div v-else-if="currentView === 'confirm_download'" class="card">
        <div class="success-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
        </div>
        <h2>Image Received</h2>
        <p class="info">A document has been uploaded and is ready for download.</p>
        <button @click="requestDownload" class="btn btn-primary btn-lg">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right:8px">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="7 10 12 15 17 10"></polyline>
            <line x1="12" y1="15" x2="12" y2="3"></line>
          </svg>
          Download PDF
        </button>
      </div>

      <!-- Waiting for phone confirmation -->
      <div v-else-if="currentView === 'waiting_confirm'" class="card">
        <div class="pulse-ring">
          <div class="pulse-ring-inner"></div>
        </div>
        <h2>Waiting for Confirmation</h2>
        <p class="info">Please confirm the download on your phone.</p>
        <p class="hint">Do not close this window.</p>
      </div>
    </div>

    <div class="footer">
      <span class="footer-badge">Session</span>
      <span class="footer-text">{{ sessionStore.sessionID?.slice(0, 8) || 'None' }}...</span>
      <span class="footer-dot"></span>
      <span class="footer-text">{{ sessionStore.status }}</span>
    </div>
  </div>
</template>

<style scoped>
.desktop-view {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}

.header {
  text-align: center;
  padding: 32px 0 16px;
}

.logo {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  background: var(--color-primary-bg);
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.3);
}

.logo-icon {
  font-size: 1.8rem;
}

.header h1 {
  font-size: 2rem;
  font-weight: 800;
  color: var(--color-text);
  margin-bottom: 6px;
  letter-spacing: -0.02em;
}

.subtitle {
  color: var(--color-text-secondary);
  font-size: 1rem;
  font-weight: 400;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px 0;
}

.card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
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

.card h2 {
  font-size: 1.4rem;
  font-weight: 700;
  color: var(--color-text);
}

.card .info {
  color: var(--color-text-secondary);
  font-size: 0.95rem;
  text-align: center;
  line-height: 1.5;
}

.success-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: linear-gradient(135deg, #10b981, #059669);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(16, 185, 129, 0.3);
}

.error-icon {
  font-size: 2.5rem;
}

.error-message {
  font-size: 1rem;
  font-weight: 500;
  color: var(--color-error);
  text-align: center;
}

.hint {
  color: var(--color-warning);
  font-size: 0.85rem;
  font-weight: 600;
  text-align: center;
}

.loading-text {
  color: var(--color-text-secondary);
  font-size: 0.95rem;
}

/* Spinner */
.spinner {
  width: 44px;
  height: 44px;
  border: 4px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Pulse animation for waiting */
.pulse-ring {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: rgba(99, 102, 241, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  animation: pulse-ring-anim 2s ease-in-out infinite;
}

.pulse-ring-inner {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: pulse-inner 2s ease-in-out infinite;
}

@keyframes pulse-ring-anim {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.15); opacity: 0.7; }
}

@keyframes pulse-inner {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 12px 28px;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.95rem;
  font-weight: 600;
  font-family: inherit;
  transition: var(--transition);
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  box-shadow: 0 4px 14px rgba(99, 102, 241, 0.35);
}

.btn-primary:hover {
  background: var(--color-primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
}

.btn-primary:active {
  transform: translateY(0);
}

.btn-primary:disabled {
  background: #94a3b8;
  box-shadow: none;
  cursor: not-allowed;
  transform: none;
}

.btn-lg {
  padding: 14px 36px;
  font-size: 1.05rem;
  border-radius: var(--radius-md);
}

/* Footer */
.footer {
  text-align: center;
  padding: 16px 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.footer-badge {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-primary);
  background: var(--color-primary-light);
  padding: 2px 8px;
  border-radius: 4px;
}

.footer-text {
  color: var(--color-text-muted);
  font-size: 0.8rem;
  font-family: 'SF Mono', 'Fira Code', monospace;
}

.footer-dot {
  width: 4px;
  height: 4px;
  background: var(--color-text-muted);
  border-radius: 50%;
}
</style>
