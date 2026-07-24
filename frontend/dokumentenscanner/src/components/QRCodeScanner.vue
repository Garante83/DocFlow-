<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { BrowserQRCodeReader } from '@zxing/library'

interface Emits {
  (e: 'scanned', data: string): void
  (e: 'back'): void
}

const emit = defineEmits<Emits>()

const videoRef = ref<HTMLVideoElement | null>(null)
const qrReader = ref<BrowserQRCodeReader | null>(null)
const isScanning = ref(false)
const errorMessage = ref<string | null>(null)
const scannedData = ref<string | null>(null)
const hasCameraAccess = ref(false)

// Initialize QR scanner
onMounted(async () => {
  await initializeScanner()
})

async function initializeScanner() {
  try {
    qrReader.value = new BrowserQRCodeReader()
    
    // Check if camera access is available
    const videoInputDevices = await qrReader.value.listVideoInputDevices()
    hasCameraAccess.value = videoInputDevices.length > 0
    
    if (hasCameraAccess.value && videoRef.value) {
      await startScanning()
    } else {
      errorMessage.value = 'No camera detected. Please use the upload option instead.'
    }
  } catch (error) {
    errorMessage.value = 'Could not access camera. Please check permissions.'
    console.error('Camera access error:', error)
  }
}

async function startScanning() {
  if (!qrReader.value || !videoRef.value) return

  isScanning.value = true
  errorMessage.value = null
  scannedData.value = null

  try {
    await qrReader.value.decodeFromVideoDevice(
      null,
      videoRef.value,
      (result: { getText(): string } | undefined, error: { name: string } | undefined) => {
        if (result) {
          handleScanResult(result.getText())
        }
        
        if (error) {
          // Ignore not found errors - these are expected when no QR code is present
          if (error.name !== 'NotFoundException') {
            console.error('Scan error:', error)
            errorMessage.value = 'Scan error. Please try again.'
          }
        }
      }
    )
  } catch (error) {
    errorMessage.value = 'Could not start camera. Please check permissions.'
    console.error('Scan start error:', error)
  }
}

function handleScanResult(data: string) {
  if (!isScanning.value) return
  
  // Stop scanning after successful read
  stopScanning()
  
  scannedData.value = data
  emit('scanned', data)
}

function stopScanning() {
  if (qrReader.value) {
    qrReader.value.reset()
  }
  isScanning.value = false
}

function handleBack() {
  stopScanning()
  emit('back')
}

// Clean up on unmount
onUnmounted(() => {
  stopScanning()
})

// Restart scanning
function restartScanning() {
  if (hasCameraAccess.value) {
    startScanning()
  }
}
</script>

<template>
  <div class="qr-scanner-container">
    <div class="header">
      <h2>Scan QR Code</h2>
      <p class="info">Point your camera at a QR code containing the session ID</p>
    </div>

    <!-- Error message -->
    <div v-if="errorMessage" class="error-message">
      <p>{{ errorMessage }}</p>
      <button @click="restartScanning" class="retry-btn" v-if="hasCameraAccess">Retry</button>
    </div>

    <!-- Video preview -->
    <div v-else class="scanner-area">
      <div class="video-container">
        <video ref="videoRef" playsinline muted class="video-preview"></video>
        <div class="scan-overlay">
          <div class="scan-frame"></div>
        </div>
      </div>
      
      <div v-if="isScanning" class="scanning-indicator">
        <div class="laser-line"></div>
        <p>Scanning...</p>
      </div>
    </div>

    <!-- Scanned result -->
    <div v-if="scannedData" class="scan-result">
      <p>Scanned: {{ scannedData }}</p>
    </div>

    <!-- Actions -->
    <div class="actions">
      <button @click="handleBack" class="btn btn-secondary">
        Back
      </button>
    </div>

    <!-- No camera access -->
    <div v-if="!hasCameraAccess && !errorMessage" class="no-camera">
      <p>No camera access. Please use the upload option instead.</p>
      <button @click="handleBack" class="btn btn-primary">Go Back</button>
    </div>
  </div>
</template>

<style scoped>
.qr-scanner-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  max-width: 500px;
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

.error-message {
  color: #d32f2f;
  text-align: center;
}

.retry-btn {
  padding: 8px 16px;
  background-color: #3498db;
  color: white;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  margin-top: 10px;
}

.scanner-area {
  position: relative;
  width: 100%;
  aspect-ratio: 4/3;
}

.video-container {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 8px;
  border: 2px solid #ddd;
}

.video-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
  background-color: #000;
}

.scan-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
}

.scan-frame {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 70%;
  height: 70%;
  border: 2px solid rgba(52, 152, 219, 0.5);
  border-radius: 8px;
}

.scanning-indicator {
  position: absolute;
  bottom: 10px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  color: #3498db;
}

.laser-line {
  width: 80%;
  height: 2px;
  background: linear-gradient(90deg, transparent, #3498db, transparent);
  animation: scan 2s linear infinite;
}

@keyframes scan {
  0% { transform: translateY(-100%); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { transform: translateY(100%); opacity: 0; }
}

.scan-result {
  padding: 15px;
  background-color: #e8f5e9;
  border-radius: 8px;
  color: #2e7d32;
  text-align: center;
  word-break: break-all;
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
}

.btn-primary {
  background-color: #3498db;
  color: white;
}

.btn-secondary {
  background-color: #ecf0f1;
  color: #2c3e50;
}

.no-camera {
  text-align: center;
  padding: 20px;
  color: #7f8c8d;
}
</style>
