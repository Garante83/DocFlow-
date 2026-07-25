<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useSessionStore } from '../stores/sessionStore'
import { apiService } from '../utils/api'
import { websocketClient } from '../utils/websocket'
import PINInput from '../components/PINInput.vue'
import ImageUpload from '../components/ImageUpload.vue'

const route = useRoute()
const sessionStore = useSessionStore()

type ViewState = 'loading' | 'error' | 'pin' | 'upload' | 'finalize' | 'confirm_download' | 'done'
const currentView = ref<ViewState>('loading')
const errorMessage = ref<string | null>(null)
const pageCount = ref(0)

function handleDownloadRequestEvent() {
  if (currentView.value === 'done' || currentView.value === 'finalize') {
    currentView.value = 'confirm_download'
  }
}

onMounted(() => {
  const sessionId = route.query.session_id as string
  if (sessionId) {
    sessionStore.setSessionID(sessionId)
    currentView.value = 'pin'
  } else {
    errorMessage.value = 'No session ID provided. Please scan the QR code again.'
    currentView.value = 'error'
  }

  websocketClient.on('download_request', handleDownloadRequestEvent)
})

onUnmounted(() => {
  websocketClient.off('download_request', handleDownloadRequestEvent)
})

function handlePINVerified() {
  currentView.value = 'upload'
  sessionStore.setStatus('upload_allowed')
}

function handlePageAdded(data: unknown) {
  const event = data as { page_count?: number }
  pageCount.value = event.page_count || sessionStore.imageCount
  sessionStore.setStatus('uploading')
  // Stay in upload view so user can add more pages
}

function handleBackToPIN() {
  currentView.value = 'pin'
}

async function handleFinalize() {
  if (!sessionStore.sessionID) return
  try {
    currentView.value = 'finalize'
    await apiService.finalizeUpload(sessionStore.sessionID)
    sessionStore.setStatus('uploaded')
  } catch (error) {
    console.error('Finalize failed:', error)
    errorMessage.value = 'Failed to generate PDF. Please try again.'
    currentView.value = 'upload'
  }
}

function confirmDownload() {
  websocketClient.send('download_confirmed', { session_id: sessionStore.sessionID })
  currentView.value = 'done'
}
</script>

<template>
  <div class="mobile-view">
    <div class="header">
      <div class="logo">
        <span class="logo-icon">&#128196;</span>
      </div>
      <h1>Doc Scanner</h1>
      <p class="subtitle">Upload your document</p>
    </div>

    <div class="main-content">
      <!-- Loading state -->
      <div v-if="currentView === 'loading'" class="card">
        <div class="spinner"></div>
        <p class="loading-text">Joining session...</p>
      </div>

      <!-- Error state -->
      <div v-else-if="currentView === 'error'" class="card">
        <div class="error-icon">&#9888;</div>
        <p class="error-message">{{ errorMessage }}</p>
      </div>

      <!-- PIN verification -->
      <PINInput
        v-else-if="currentView === 'pin'"
        @verified="handlePINVerified"
        @back="() => {}"
      />

      <!-- Image upload -->
      <ImageUpload
        v-else-if="currentView === 'upload'"
        @page-added="handlePageAdded"
        @back="handleBackToPIN"
      />

      <!-- Finalize: generating PDF -->
      <div v-else-if="currentView === 'finalize'" class="card">
        <div class="spinner"></div>
        <h2>Generating PDF</h2>
        <p class="info">Processing {{ pageCount }} {{ pageCount === 1 ? 'page' : 'pages' }}...</p>
      </div>

      <!-- Download confirmation -->
      <div v-else-if="currentView === 'confirm_download'" class="card">
        <div class="download-icon-wrap">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="7 10 12 15 17 10"></polyline>
            <line x1="12" y1="15" x2="12" y2="3"></line>
          </svg>
        </div>
        <h2>Download Requested</h2>
        <p class="info">The desktop wants to download the PDF.</p>
        <button @click="confirmDownload" class="btn btn-success btn-full">
          Confirm Download
        </button>
        <button @click="currentView = 'done'" class="btn btn-ghost btn-full">
          Cancel
        </button>
        <p class="hint">Do not close this window until the download has started on the desktop.</p>
      </div>

      <!-- Upload complete -->
      <div v-else-if="currentView === 'done'" class="card">
        <div class="success-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
        </div>
        <h2>Upload Complete</h2>
        <p class="info">Your document has been sent. You can close this page.</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mobile-view {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  width: 100%;
  padding: 12px;
}

.header {
  text-align: center;
  padding: 20px 0 8px;
}

.logo {
  width: 48px;
  height: 48px;
  margin: 0 auto 10px;
  background: var(--color-primary-bg);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 6px 18px rgba(99, 102, 241, 0.3);
}

.logo-icon {
  font-size: 1.3rem;
}

.header h1 {
  font-size: 1.4rem;
  font-weight: 800;
  color: var(--color-text);
  margin-bottom: 3px;
  letter-spacing: -0.02em;
}

.subtitle {
  color: var(--color-text-secondary);
  font-size: 0.85rem;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 16px 0;
}

.card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  max-width: 420px;
  width: 100%;
  padding: 32px 24px;
  background: var(--color-surface);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  text-align: center;
}

.card h2 {
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
}

.card .info {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  margin: 0;
  line-height: 1.5;
}

.success-icon {
  width: 68px;
  height: 68px;
  border-radius: 50%;
  background: linear-gradient(135deg, #10b981, #059669);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(16, 185, 129, 0.3);
}

.download-icon-wrap {
  width: 68px;
  height: 68px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.3);
}

.error-icon {
  font-size: 2.2rem;
}

.error-message {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--color-error);
  text-align: center;
}

.loading-text {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.hint {
  color: var(--color-warning);
  font-size: 0.78rem;
  font-weight: 500;
  text-align: center;
  margin: 0;
  line-height: 1.4;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 12px 24px;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.95rem;
  font-weight: 600;
  font-family: inherit;
  transition: var(--transition);
  width: 100%;
}

.btn-full { width: 100%; }

.btn-success {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
}

.btn-success:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.4);
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.btn-ghost:hover {
  background: var(--color-border);
}
</style>
