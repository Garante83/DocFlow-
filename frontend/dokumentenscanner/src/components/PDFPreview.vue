<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '../stores/sessionStore'
import { apiService } from '../utils/api'

const { t } = useI18n()

interface Emits {
  (e: 'downloaded'): void
  (e: 'back'): void
}

const emit = defineEmits<Emits>()
const sessionStore = useSessionStore()

const pdfBlob = ref<Blob | null>(null)
const pdfUrl = ref<string | null>(null)
const isLoading = ref(true)
const errorMessage = ref<string | null>(null)
const pollInterval = ref<number | null>(null)

// Initialize and fetch PDF
onMounted(async () => {
  await fetchPDF()
  
  // Set up polling for PDF status
  startPolling()
})

// Clean up on unmount
onUnmounted(() => {
  stopPolling()
  cleanupPDF()
})

async function fetchPDF() {
  if (!sessionStore.sessionID) {
    errorMessage.value = t('pdf.noSessionId')
    isLoading.value = false
    return
  }

  try {
    await downloadPDF()
  } catch (error) {
    console.error('Error fetching PDF:', error)
    errorMessage.value = t('pdf.loadFailed')
    isLoading.value = false
  }
}

async function downloadPDF() {
  if (!sessionStore.sessionID) return

  try {
    const blob = await apiService.downloadPDF(sessionStore.sessionID)
    pdfBlob.value = blob
    pdfUrl.value = URL.createObjectURL(blob)
    
    // Store in session
    sessionStore.setPDF(blob)
    sessionStore.setStatus('ready')
    
    errorMessage.value = null
    isLoading.value = false
  } catch (error) {
    console.error('Error downloading PDF:', error)
    errorMessage.value = t('pdf.downloadFailed')
    isLoading.value = false
  }
}

// Poll for PDF status
function startPolling() {
  pollInterval.value = window.setInterval(async () => {
    if (!sessionStore.sessionID) {
      stopPolling()
      return
    }

    try {
      await downloadPDF()
      stopPolling()
    } catch {
      // PDF not ready yet, continue polling
    }
  }, 2000) // Poll every 2 seconds
}

function stopPolling() {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = null
  }
}

// Download the PDF file
function downloadPDFFile() {
  if (!pdfBlob.value) return

  const url = URL.createObjectURL(pdfBlob.value)
  const a = document.createElement('a')
  a.href = url
  a.download = `document-${sessionStore.sessionID || 'unknown'}.pdf`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  
  emit('downloaded')
}

// Clean up PDF URL
function cleanupPDF() {
  if (pdfUrl.value) {
    URL.revokeObjectURL(pdfUrl.value)
    pdfUrl.value = null
  }
}

function handleBack() {
  cleanupPDF()
  emit('back')
}

// Retry downloading
async function retryDownload() {
  isLoading.value = true
  errorMessage.value = null
  await fetchPDF()
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 ' + t('pdf.units.bytes')
  
  const k = 1024
  const sizes = [t('pdf.units.bytes'), t('pdf.units.kb'), t('pdf.units.mb'), t('pdf.units.gb')]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>

<template>
  <div class="pdf-preview-container">
    <div class="header">
      <h2>{{ t('pdf.preview') }}</h2>
      <p class="info">{{ t('pdf.converted') }}</p>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="loading">
      <div class="spinner"></div>
      <p>{{ t('pdf.generating') }}</p>
    </div>

    <!-- Error state -->
    <div v-else-if="errorMessage" class="error">
      <p class="error-message">{{ errorMessage }}</p>
      <button @click="retryDownload" class="retry-btn">{{ t('common.retry') }}</button>
    </div>

    <!-- PDF preview -->
    <div v-else-if="pdfUrl" class="pdf-content">
      <div class="pdf-viewer">
        <iframe
          :src="pdfUrl"
          class="pdf-iframe"
          frameborder="0"
        ></iframe>
      </div>
      
      <div class="file-info">
        <p>{{ t('pdf.document') }}</p>
        <p v-if="pdfBlob" class="file-size">
          {{ formatFileSize(pdfBlob.size) }}
        </p>
      </div>

      <!-- Actions -->
      <div class="actions">
        <button @click="handleBack" class="btn btn-secondary">
          {{ t('common.back') }}
        </button>
        <button @click="downloadPDFFile" class="btn btn-primary">
          {{ t('pdf.downloadPdf') }}
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="empty-state">
      <p>{{ t('pdf.noPdf') }}</p>
      <button @click="retryDownload" class="btn btn-primary">{{ t('common.retry') }}</button>
    </div>
  </div>
</template>

<style scoped>
.pdf-preview-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  max-width: 800px;
  width: 100%;
  padding: 20px;
  background: white;
  border-radius: 10px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.header {
  text-align: center;
}

.header h2 {
  font-size: 1.5rem;
  color: var(--color-text);
  margin-bottom: 5px;
}

.info {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
  padding: 40px 0;
}

.spinner {
  width: 50px;
  height: 50px;
  border: 5px solid var(--color-border);
  border-top: 5px solid var(--color-info);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
  color: var(--color-error);
}

.error-message {
  font-size: 1rem;
  font-weight: 500;
}

.retry-btn {
  padding: 10px 20px;
  background-color: var(--color-info);
  color: white;
  border: none;
  border-radius: 5px;
  cursor: pointer;
}

.pdf-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.pdf-viewer {
  width: 100%;
  height: 400px;
  border: 2px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
  background: white;
}

.pdf-iframe {
  width: 100%;
  height: 100%;
  border: none;
}

.file-info {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.file-size {
  font-weight: 500;
}

.actions {
  display: flex;
  gap: 15px;
  width: 100%;
  margin-top: 10px;
}

.btn {
  flex: 1;
  padding: 12px 20px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 500;
  transition: all 0.3s;
}

.btn-primary {
  background-color: #2e7d32;
  color: white;
}

.btn-secondary {
  background-color: var(--color-surface-muted);
  color: var(--color-text);
}

.btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
  padding: 40px 0;
  color: var(--color-text-secondary);
  text-align: center;
}
</style>
