<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useSessionStore } from '../stores/sessionStore'
import { apiService } from '../utils/api'

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
    errorMessage.value = 'No session ID available'
    isLoading.value = false
    return
  }

  try {
    await downloadPDF()
  } catch (error) {
    console.error('Error fetching PDF:', error)
    errorMessage.value = 'Failed to load PDF. Please try again.'
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
    errorMessage.value = 'Failed to download PDF. Please try again.'
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
  if (bytes === 0) return '0 Bytes'
  
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>

<template>
  <div class="pdf-preview-container">
    <div class="header">
      <h2>PDF Preview</h2>
      <p class="info">Your document has been converted to PDF</p>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="loading">
      <div class="spinner"></div>
      <p>Generating PDF...</p>
    </div>

    <!-- Error state -->
    <div v-else-if="errorMessage" class="error">
      <p class="error-message">{{ errorMessage }}</p>
      <button @click="retryDownload" class="retry-btn">Retry</button>
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
        <p>PDF Document</p>
        <p v-if="pdfBlob" class="file-size">
          {{ formatFileSize(pdfBlob.size) }}
        </p>
      </div>

      <!-- Actions -->
      <div class="actions">
        <button @click="handleBack" class="btn btn-secondary">
          Back
        </button>
        <button @click="downloadPDFFile" class="btn btn-primary">
          Download PDF
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="empty-state">
      <p>No PDF available</p>
      <button @click="retryDownload" class="btn btn-primary">Retry</button>
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
  color: #2c3e50;
  margin-bottom: 5px;
}

.info {
  color: #7f8c8d;
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
  border: 5px solid #f3f3f3;
  border-top: 5px solid #3498db;
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
  color: #d32f2f;
}

.error-message {
  font-size: 1rem;
  font-weight: 500;
}

.retry-btn {
  padding: 10px 20px;
  background-color: #3498db;
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
  border: 2px solid #ddd;
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
  color: #7f8c8d;
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
  background-color: #ecf0f1;
  color: #2c3e50;
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
  color: #7f8c8d;
  text-align: center;
}
</style>
