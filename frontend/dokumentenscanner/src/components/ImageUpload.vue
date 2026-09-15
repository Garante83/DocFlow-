<script setup lang="ts">
import { ref, computed, onUnmounted, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '../stores/sessionStore'
import { apiService } from '../utils/api'
import { AngleIndicator, tiltStatus } from '../utils/angleIndicator'
import { FrameAnalyzer } from '../utils/frameAnalyzer'
import type { FrameStatus } from '../utils/frameAnalyzer'

interface Emits {
  (e: 'pageAdded', data: { page_count: number }): void
  (e: 'finalize'): void
  (e: 'back'): void
}

const emit = defineEmits<Emits>()
const { t } = useI18n()
const sessionStore = useSessionStore()

type Mode = 'choose' | 'camera' | 'edit'
const mode = ref<Mode>('choose')

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const previewUrl = ref<string | null>(null)
const isUploading = ref(false)
const errorMessage = ref<string | null>(null)
const successMessage = ref<string | null>(null)

// Camera
const videoRef = ref<HTMLVideoElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const cameraStream = ref<MediaStream | null>(null)
const cameraReady = ref(false)

// Tilt indicator: sensor chain first (deviceorientation -> Accelerometer),
// then visual fallback (video frame analysis) when sensors are unavailable.
const angleIndicator = new AngleIndicator()
const frameAnalyzer = new FrameAnalyzer()
const tiltAvailable = ref(false)
const tiltStatusName = ref<'good' | 'ok' | 'bad' | 'filled'>('good')
const angleTextKey = computed(() => `upload.angle${tiltStatusName.value.charAt(0).toUpperCase()}${tiltStatusName.value.slice(1)}`)

let visualPending = false

function onSensorUpdate(deg: number) {
  tiltStatusName.value = tiltStatus(deg)
  tiltAvailable.value = true
}

function onVisualStatus(status: FrameStatus) {
  if (status === 'nodoc') {
    tiltAvailable.value = false
    return
  }
  tiltStatusName.value = status
  tiltAvailable.value = true
}

function startVisualFallback() {
  const video = videoRef.value
  if (video === null || video.videoWidth === 0) {
    visualPending = true
    return
  }
  visualPending = false
  frameAnalyzer.start(video, onVisualStatus)
}

watch(cameraReady, (ready) => {
  if (ready && visualPending) startVisualFallback()
})

// Rotation
const rotation = ref(0)

// Crop
const showCrop = ref(false)
const cropCanvasRef = ref<HTMLCanvasElement | null>(null)
const cropImageBitmap = ref<ImageBitmap | null>(null)
const cropArea = ref({ x: 0, y: 0, w: 0, h: 0 })
const canvasScale = ref(1)
const isDragging = ref(false)
const dragCorner = ref<string | null>(null)
const dragOffset = ref({ x: 0, y: 0 })

const canUpload = computed(() => selectedFile.value !== null && !isUploading.value && !errorMessage.value)

// Edit index: null = new image, number = editing existing image in the list
const editIndex = ref<number | null>(null)

// Pre-compute blob URLs for page thumbnails (avoids Memory Leak from URL.createObjectURL in template)
const pageThumbs = computed(() => {
  return sessionStore.images.map((file) => URL.createObjectURL(file))
})

// Compute total upload size in MB
const totalUploadMB = computed(() => {
  const bytes = sessionStore.images.reduce((sum, file) => sum + file.size, 0)
  return (bytes / (1024 * 1024)).toFixed(1)
})

const uploadPercent = computed(() => {
  return Math.min(100, Math.round((parseFloat(totalUploadMB.value) / sessionStore.maxFileSizeMB) * 100))
})

async function loadImageBitmap(file: File): Promise<ImageBitmap> {
  return createImageBitmap(file, { imageOrientation: 'from-image' })
}

// Edit from list: load an existing image into edit mode
function editFromList(index: number) {
  if (index < 0 || index >= sessionStore.images.length) return
  const raw = sessionStore.images[index]
  if (raw === undefined) return
  const file = toRaw(raw) as File
  editIndex.value = index
  selectedFile.value = file
  previewUrl.value = URL.createObjectURL(file)
  rotation.value = 0
  isUploading.value = false
  errorMessage.value = null
  successMessage.value = null
  mode.value = 'edit'
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files || files.length === 0) return

  const validTypes = ['image/jpeg', 'image/png', 'image/jpg']

  // If only one file, show it in edit mode
  if (files.length === 1) {
    const file = files[0]
    if (!file) return
    if (!validTypes.includes(file.type)) {
      errorMessage.value = t('upload.invalidType', { name: file.name })
      return
    }
    if (file.size > sessionStore.maxFileSizeMB * 1024 * 1024) {
      errorMessage.value = t('upload.tooLarge', { name: file.name, size: formatFileSize(file.size), max: sessionStore.maxFileSizeMB })
      return
    }
    selectedFile.value = file
    errorMessage.value = null
    successMessage.value = null
    isUploading.value = false
    previewUrl.value = URL.createObjectURL(file)
    rotation.value = 0
    mode.value = 'edit'
    return
  }

  // Multiple files: upload all directly
  uploadMultipleFiles(Array.from(files))
}

async function uploadMultipleFiles(files: File[]) {
  if (!sessionStore.sessionID) return

  isUploading.value = true
  errorMessage.value = null
  successMessage.value = null

  const validTypes = ['image/jpeg', 'image/png', 'image/jpg']
  let uploaded = 0
  let failed = 0
  let lastError = ''

  try {
    for (const file of files) {
      if (!validTypes.includes(file.type)) {
        lastError = t('upload.invalidType', { name: file.name })
        failed++
        continue
      }
      if (file.size > sessionStore.maxFileSizeMB * 1024 * 1024) {
        lastError = t('upload.tooLarge', { name: file.name, size: formatFileSize(file.size), max: sessionStore.maxFileSizeMB })
        failed++
        continue
      }
      try {
        const normalized = await normalizeImage(file, 0)
        await apiService.uploadImage(sessionStore.sessionID, normalized)
        sessionStore.addImage(normalized)
        uploaded++
      } catch (error: unknown) {
        const err = error as { userMessage?: string; response?: { data?: { error?: string } } }
        lastError = err.userMessage || err.response?.data?.error || t('upload.uploadFailed')
        failed++
      }
    }

    if (uploaded > 0) {
      sessionStore.setStatus('uploading')
      emit('pageAdded', { page_count: sessionStore.imageCount })
      successMessage.value = t('upload.pagesAdded', { count: uploaded })
    }
    if (failed > 0) {
      errorMessage.value = lastError || t('upload.filesFailed', { count: failed })
    }
  } finally {
    isUploading.value = false
  }
}

function triggerFileInput() {
  fileInput.value?.click()
}

// ==================== CAMERA ====================

async function openCamera() {
  errorMessage.value = null
  try {
    // Request sensor permission FIRST (synchronous part of the click gesture),
    // before getUserMedia - Safari drops the gesture context after the first await.
    const sensorOk = await angleIndicator.requestPermission()
    tiltAvailable.value = false
    visualPending = false
    if (sensorOk) {
      angleIndicator.start(onSensorUpdate, () => startVisualFallback())
    } else {
      // Permission denied on iOS -> go straight to the visual fallback
      startVisualFallback()
    }

    const stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment', width: { ideal: 1920 }, height: { ideal: 1080 } },
      audio: false,
    })
    cameraStream.value = stream
    mode.value = 'camera'
    cameraReady.value = false

    setTimeout(() => {
      if (videoRef.value) {
        videoRef.value.srcObject = stream
        videoRef.value.play()
        videoRef.value.onloadedmetadata = () => {
          cameraReady.value = true
        }
      }
    }, 100)
  } catch (err) {
    console.error('Camera error:', err)
    angleIndicator.stop()
    frameAnalyzer.stop()
    visualPending = false
    tiltAvailable.value = false
    errorMessage.value = t('upload.cameraError')
  }
}

function closeCamera() {
  if (cameraStream.value) {
    cameraStream.value.getTracks().forEach(t => t.stop())
    cameraStream.value = null
  }
  angleIndicator.stop()
  frameAnalyzer.stop()
  visualPending = false
  tiltAvailable.value = false
  mode.value = 'choose'
}

function capturePhoto() {
  if (!videoRef.value || !canvasRef.value) return
  const video = videoRef.value
  const canvas = canvasRef.value
  canvas.width = video.videoWidth
  canvas.height = video.videoHeight
  const ctx = canvas.getContext('2d')!
  ctx.drawImage(video, 0, 0)

  canvas.toBlob((blob) => {
    if (!blob) return
    const file = new File([blob], 'capture.jpg', { type: 'image/jpeg' })
    selectedFile.value = file
    previewUrl.value = URL.createObjectURL(blob)
    rotation.value = 0
    isUploading.value = false
    errorMessage.value = null
    successMessage.value = null
    closeCamera()
    mode.value = 'edit'
  }, 'image/jpeg', 0.92)
}

// ==================== ROTATION ====================

function rotateLeft() {
  rotation.value = (rotation.value - 90 + 360) % 360
}

function rotateRight() {
  rotation.value = (rotation.value + 90) % 360
}

// ==================== CROP ====================

async function openCrop() {
  if (!selectedFile.value) return
  showCrop.value = true
  try {
    const bmp = await loadImageBitmap(selectedFile.value)
    cropImageBitmap.value = bmp
    // Place the crop frame visibly inside the image (5% inset, centered)
    const insetX = bmp.width * 0.05
    const insetY = bmp.height * 0.05
    cropArea.value = { x: insetX, y: insetY, w: bmp.width - insetX * 2, h: bmp.height - insetY * 2 }
    drawCrop()
  } catch {
    showCrop.value = false
    errorMessage.value = t('upload.cropError')
  }
}

function drawCrop() {
  const canvas = cropCanvasRef.value
  const bmp = cropImageBitmap.value
  if (!canvas || !bmp) return

  const containerWidth = canvas.parentElement?.clientWidth || 350
  const scale = containerWidth / bmp.width
  canvasScale.value = scale
  canvas.width = containerWidth
  canvas.height = bmp.height * scale

  const ctx = canvas.getContext('2d')!
  ctx.clearRect(0, 0, canvas.width, canvas.height)

  // Draw image
  ctx.drawImage(bmp, 0, 0, canvas.width, canvas.height)

  // Dark overlay outside crop
  const c = cropArea.value
  const sx = c.x * scale
  const sy = c.y * scale
  const sw = c.w * scale
  const sh = c.h * scale

  ctx.fillStyle = 'rgba(0, 0, 0, 0.5)'
  ctx.fillRect(0, 0, canvas.width, canvas.height)

  // Clear crop region and redraw image there
  ctx.save()
  ctx.beginPath()
  ctx.rect(sx, sy, sw, sh)
  ctx.clip()
  ctx.drawImage(bmp, 0, 0, canvas.width, canvas.height)
  ctx.restore()

  // Rule-of-thirds grid inside the crop area
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.3)'
  ctx.lineWidth = 1
  ctx.beginPath()
  for (let i = 1; i <= 2; i++) {
    const gx = sx + (sw / 3) * i
    const gy = sy + (sh / 3) * i
    ctx.moveTo(gx, sy)
    ctx.lineTo(gx, sy + sh)
    ctx.moveTo(sx, gy)
    ctx.lineTo(sx + sw, gy)
  }
  ctx.stroke()

  // Border: dark contour under white line for contrast on light backgrounds
  ctx.strokeStyle = 'rgba(0, 0, 0, 0.5)'
  ctx.lineWidth = 5
  ctx.strokeRect(sx, sy, sw, sh)
  ctx.strokeStyle = '#fff'
  ctx.lineWidth = 3
  ctx.strokeRect(sx, sy, sw, sh)

  // Corner handles (clamped fully inside the canvas)
  const handleSize = 16
  ctx.fillStyle = '#3498db'
  ctx.strokeStyle = '#fff'
  ctx.lineWidth = 1
  const corners: [number, number][] = [
    [sx, sy], [sx + sw, sy], [sx, sy + sh], [sx + sw, sy + sh],
  ]
  for (const [cx, cy] of corners) {
    const hx = Math.max(handleSize / 2, Math.min(cx, canvas.width - handleSize / 2))
    const hy = Math.max(handleSize / 2, Math.min(cy, canvas.height - handleSize / 2))
    ctx.fillRect(hx - handleSize / 2, hy - handleSize / 2, handleSize, handleSize)
    ctx.strokeRect(hx - handleSize / 2, hy - handleSize / 2, handleSize, handleSize)
  }
}

function getEventCoords(e: TouchEvent | MouseEvent): { x: number; y: number } {
  const canvas = cropCanvasRef.value!
  const rect = canvas.getBoundingClientRect()
  let clientX: number, clientY: number
  if ('touches' in e) {
    const touch = e.touches[0]
    if (!touch) return { x: 0, y: 0 }
    clientX = touch.clientX
    clientY = touch.clientY
  } else {
    clientX = e.clientX
    clientY = e.clientY
  }
  // Convert to canvas pixel coordinates
  const canvasX = (clientX - rect.left) * (canvas.width / rect.width)
  const canvasY = (clientY - rect.top) * (canvas.height / rect.height)
  // Convert to image coordinates
  return { x: canvasX / canvasScale.value, y: canvasY / canvasScale.value }
}

function hitTest(coords: { x: number; y: number }): string | null {
  const c = cropArea.value
  const threshold = 25 / canvasScale.value

  const corners: [string, number, number][] = [
    ['tl', c.x, c.y],
    ['tr', c.x + c.w, c.y],
    ['bl', c.x, c.y + c.h],
    ['br', c.x + c.w, c.y + c.h],
  ]

  for (const [name, cx, cy] of corners) {
    if (Math.abs(coords.x - cx) < threshold && Math.abs(coords.y - cy) < threshold) {
      return name
    }
  }

  // Check if inside crop area (for moving)
  if (coords.x >= c.x && coords.x <= c.x + c.w && coords.y >= c.y && coords.y <= c.y + c.h) {
    return 'move'
  }

  return null
}

function onCropStart(e: TouchEvent | MouseEvent) {
  e.preventDefault()
  const coords = getEventCoords(e)
  const hit = hitTest(coords)
  if (!hit) return

  isDragging.value = true
  dragCorner.value = hit

  const c = cropArea.value
  if (hit === 'move') {
    dragOffset.value = { x: coords.x - c.x, y: coords.y - c.y }
  } else {
    dragOffset.value = coords
  }
}

function onCropMove(e: TouchEvent | MouseEvent) {
  if (!isDragging.value || !dragCorner.value) return
  e.preventDefault()

  const coords = getEventCoords(e)
  const bmp = cropImageBitmap.value!
  const c = cropArea.value
  const minSize = 30

  if (dragCorner.value === 'move') {
    const newX = Math.max(0, Math.min(coords.x - dragOffset.value.x, bmp.width - c.w))
    const newY = Math.max(0, Math.min(coords.y - dragOffset.value.y, bmp.height - c.h))
    c.x = newX
    c.y = newY
  } else {
    const corner = dragCorner.value
    // Each corner adjusts its respective edges
    if (corner.includes('l')) {
      const newX = Math.max(0, Math.min(coords.x, c.x + c.w - minSize))
      c.w += c.x - newX
      c.x = newX
    }
    if (corner.includes('r')) {
      c.w = Math.max(minSize, Math.min(coords.x - c.x, bmp.width - c.x))
    }
    if (corner.includes('t')) {
      const newY = Math.max(0, Math.min(coords.y, c.y + c.h - minSize))
      c.h += c.y - newY
      c.y = newY
    }
    if (corner.includes('b')) {
      c.h = Math.max(minSize, Math.min(coords.y - c.y, bmp.height - c.y))
    }
  }
  drawCrop()
}

function onCropEnd() {
  isDragging.value = false
  dragCorner.value = null
}

function applyCrop() {
  const bmp = cropImageBitmap.value
  if (!bmp) return
  const c = cropArea.value

  const offscreen = document.createElement('canvas')
  offscreen.width = c.w
  offscreen.height = c.h
  const ctx = offscreen.getContext('2d')!
  ctx.drawImage(bmp, c.x, c.y, c.w, c.h, 0, 0, c.w, c.h)

  offscreen.toBlob((blob) => {
    if (!blob) return
    if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
    const file = new File([blob], 'cropped.jpg', { type: 'image/jpeg' })
    selectedFile.value = file
    previewUrl.value = URL.createObjectURL(blob)
    showCrop.value = false
    cropImageBitmap.value = null
  }, 'image/jpeg', 0.92)
}

function cancelCrop() {
  showCrop.value = false
  cropImageBitmap.value = null
}

// ==================== EDIT ACTIONS ====================

function resetEdit() {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  selectedFile.value = null
  previewUrl.value = null
  editIndex.value = null
  rotation.value = 0
  isUploading.value = false
  showCrop.value = false
  mode.value = 'choose'
}

// ==================== UPLOAD ====================

// Normalize image through canvas to strip EXIF orientation and apply rotation
// This ensures correct pixel orientation in the uploaded JPEG
async function normalizeImage(file: File, degrees: number): Promise<File> {
  const bmp = await loadImageBitmap(file)
  return new Promise((resolve) => {
    const canvas = document.createElement('canvas')
    if (degrees === 90 || degrees === 270) {
      canvas.width = bmp.height
      canvas.height = bmp.width
    } else {
      canvas.width = bmp.width
      canvas.height = bmp.height
    }
    const ctx = canvas.getContext('2d')!
    ctx.translate(canvas.width / 2, canvas.height / 2)
    ctx.rotate((degrees * Math.PI) / 180)
    ctx.drawImage(bmp, -bmp.width / 2, -bmp.height / 2)

    canvas.toBlob((blob) => {
      if (!blob) { resolve(file); return }
      resolve(new File([blob], file.name, { type: 'image/jpeg' }))
    }, 'image/jpeg', 0.92)
  })
}

async function uploadFile() {
  if (!selectedFile.value || !sessionStore.sessionID) return

  isUploading.value = true
  errorMessage.value = null
  successMessage.value = null

  try {
    // Always normalize through canvas to strip EXIF orientation
    let fileToUpload = selectedFile.value
    fileToUpload = await normalizeImage(fileToUpload, rotation.value)

    // Client-side size check with specific message
    const maxBytes = sessionStore.maxFileSizeMB * 1024 * 1024
    if (fileToUpload.size > maxBytes) {
      errorMessage.value = t('upload.tooLarge', { name: selectedFile.value.name, size: formatFileSize(fileToUpload.size), max: sessionStore.maxFileSizeMB })
      isUploading.value = false
      return
    }

    const response = await apiService.uploadImage(sessionStore.sessionID, fileToUpload)

    if (editIndex.value !== null) {
      // Replace existing image in the list
      sessionStore.images[editIndex.value] = fileToUpload
      successMessage.value = t('upload.pageUpdated')
    } else {
      // Add new image
      sessionStore.addImage(fileToUpload)
      successMessage.value = t('upload.pageAdded')
    }

    sessionStore.setStatus('uploading')
    emit('pageAdded', { page_count: response.page_count || sessionStore.imageCount })

    // Reset to choose mode for next page
    if (previewUrl.value) {
      URL.revokeObjectURL(previewUrl.value)
      previewUrl.value = null
    }
    selectedFile.value = null
    editIndex.value = null
    rotation.value = 0
    showCrop.value = false
    mode.value = 'choose'
  } catch (error: unknown) {
    const err = error as { userMessage?: string; response?: { status?: number; data?: { error?: string } } }
    
    // Use the specific error message from the API interceptor
    if (err.userMessage) {
      errorMessage.value = err.userMessage
    } else if (err.response?.data?.error) {
      errorMessage.value = err.response.data.error
    } else {
      errorMessage.value = t('upload.uploadFailed')
    }
    console.error('Upload error:', error)
  } finally {
    isUploading.value = false
  }
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 ' + t('pdf.units.bytes')
  const k = 1024
  const sizes = [t('pdf.units.bytes'), t('pdf.units.kb'), t('pdf.units.mb'), t('pdf.units.gb')]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onUnmounted(() => {
  closeCamera()
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  // Revoke pre-computed blob URLs
  pageThumbs.value.forEach((url) => URL.revokeObjectURL(url))
})
</script>

<template>
  <div class="upload-container">
    <!-- ==================== CHOOSE MODE ==================== -->
    <template v-if="mode === 'choose'">
      <div class="header">
        <h2>{{ t('upload.heading') }}</h2>
        <p class="info" v-if="sessionStore.imageCount === 0">{{ t('upload.takePhotoOrSelect') }}</p>
        <p class="info" v-else>{{ sessionStore.imageCount }} {{ sessionStore.imageCount === 1 ? t('upload.page') : t('upload.pages') }} {{ t('upload.added') }}</p>
        <div v-if="sessionStore.imageCount > 0" class="usage-bar">
          <div class="usage-fill" :style="{ width: uploadPercent + '%' }" :class="{ 'usage-warning': uploadPercent > 80 }"></div>
        </div>
        <p class="limit-info">{{ totalUploadMB }}MB / {{ sessionStore.maxFileSizeMB }}MB · {{ sessionStore.imageCount }}/{{ sessionStore.maxPages }} pages</p>
      </div>

      <div v-if="successMessage" class="success-message">{{ successMessage }}</div>

      <div class="choose-actions">
        <button @click="openCamera" class="choose-btn camera-btn">
          <span class="choose-icon-wrap camera-icon-wrap">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path>
              <circle cx="12" cy="13" r="4"></circle>
            </svg>
          </span>
          <span class="choose-label">{{ t('upload.takePhoto') }}</span>
        </button>
        <button @click="triggerFileInput" class="choose-btn file-btn">
          <span class="choose-icon-wrap file-icon-wrap">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
              <polyline points="14 2 14 8 20 8"></polyline>
            </svg>
          </span>
          <span class="choose-label">{{ t('upload.chooseFile') }}</span>
        </button>
      </div>

      <input
        ref="fileInput"
        type="file"
        accept="image/jpeg,image/png,image/jpg"
        multiple
        @change="handleFileChange"
        class="file-input"
      />

      <div v-if="errorMessage" class="error-message">{{ errorMessage }}</div>

      <!-- Page list -->
      <div v-if="sessionStore.imageCount > 0" class="page-list">
        <h3 class="page-list-title">{{ sessionStore.imageCount }} {{ t('upload.pageListTitle') }}</h3>
        <div class="page-items">
          <div v-for="(file, index) in sessionStore.images" :key="index" class="page-item" @click="editFromList(index)">
            <img :src="pageThumbs[index]" class="page-thumb" />
            <span class="page-number">{{ index + 1 }}</span>
            <button @click.stop="sessionStore.removeImage(index)" class="page-remove" :title="t('upload.removePage')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- Finalize button -->
      <div v-if="sessionStore.imageCount > 0" class="finalize-section">
        <button @click="$emit('finalize')" class="btn btn-success btn-full">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right:8px">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          {{ t('upload.generatePdf') }}
        </button>
      </div>
    </template>

    <!-- ==================== CAMERA MODE ==================== -->
    <template v-if="mode === 'camera'">
      <div class="header">
        <h2>{{ t('upload.camera') }}</h2>
        <p class="info">{{ t('upload.cameraInfo') }}</p>
      </div>

      <div class="camera-container">
        <video
          ref="videoRef"
          autoplay
          playsinline
          class="camera-video"
          :class="tiltAvailable ? `tilt-${tiltStatusName}` : ''"
        ></video>
        <canvas ref="canvasRef" class="camera-canvas-hidden"></canvas>
        <div v-if="tiltAvailable" class="angle-hud">
          <span class="angle-dot" :class="`dot-${tiltStatusName}`"></span>
          <span class="angle-text">{{ t(angleTextKey) }}</span>
        </div>
        <div v-if="!cameraReady" class="camera-loading">
          <div class="spinner"></div>
          <p>{{ t('upload.startingCamera') }}</p>
        </div>
      </div>

      <div class="camera-controls">
        <button @click="closeCamera" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button @click="capturePhoto" class="capture-btn" :disabled="!cameraReady">
          <span class="capture-ring"></span>
        </button>
        <div style="width: 80px"></div>
      </div>
    </template>

    <!-- ==================== EDIT MODE ==================== -->
    <template v-if="mode === 'edit' && !showCrop">
      <div class="header">
        <h2>{{ t('upload.editDocument') }}</h2>
        <p class="info">{{ t('upload.editInfo') }}</p>
      </div>

      <div class="preview-section" v-if="previewUrl">
        <div class="preview-image-container" :style="{ transform: `rotate(${rotation}deg)` }">
          <img :src="previewUrl" :alt="t('upload.preview')" class="preview-image" />
        </div>
        <p class="file-info">
          {{ selectedFile?.name }} ({{ formatFileSize(selectedFile?.size || 0) }})
        </p>
      </div>

      <div class="edit-toolbar">
        <button @click="rotateLeft" class="toolbar-btn" :title="t('upload.rotateLeft')">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"></polyline><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path></svg>
        </button>
        <button @click="rotateRight" class="toolbar-btn" :title="t('upload.rotateRight')">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"></polyline><path d="M20.49 15a9 9 0 1 1-2.13-9.36L23 10"></path></svg>
        </button>
        <button @click="openCrop" class="toolbar-btn" :title="t('upload.crop')">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6.13 1L6 16a2 2 0 0 0 2 2h15"></path><path d="M1 6.13L16 6a2 2 0 0 1 2 2v15"></path></svg>
        </button>
      </div>

      <div v-if="errorMessage" class="error-message">{{ errorMessage }}</div>
      <div v-if="successMessage" class="success-message">{{ successMessage }}</div>

      <div class="actions">
        <button @click="resetEdit" class="btn btn-secondary" :disabled="isUploading">{{ t('common.back') }}</button>
        <button @click="uploadFile" class="btn btn-primary" :disabled="!canUpload">
          <span v-if="isUploading">{{ t('common.uploading') }}</span>
          <span v-else>{{ editIndex !== null ? t('upload.updatePage') : t('upload.addPage') }}</span>
        </button>
      </div>
    </template>

    <!-- ==================== CROP MODE ==================== -->
    <template v-if="showCrop">
      <div class="header">
        <h2>{{ t('upload.cropDocument') }}</h2>
        <p class="info">{{ t('upload.cropInfo') }}</p>
      </div>

      <div class="crop-container">
        <canvas
          ref="cropCanvasRef"
          class="crop-canvas"
          @mousedown="onCropStart"
          @mousemove="onCropMove"
          @mouseup="onCropEnd"
          @mouseleave="onCropEnd"
          @touchstart.prevent="onCropStart"
          @touchmove.prevent="onCropMove"
          @touchend.prevent="onCropEnd"
        ></canvas>
      </div>

      <div class="actions">
        <button @click="cancelCrop" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button @click="applyCrop" class="btn btn-primary">{{ t('upload.applyCrop') }}</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.upload-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  max-width: 420px;
  width: 100%;
  padding: 32px 24px;
  background: var(--color-surface);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
}

.header { text-align: center; }
.header h2 { font-size: 1.4rem; font-weight: 700; color: var(--color-text); margin-bottom: 4px; }
.info { color: var(--color-text-secondary); font-size: 0.9rem; }
.limit-info { color: var(--color-text-muted); font-size: 0.75rem; font-weight: 500; }
.usage-bar { width: 100%; height: 6px; background: var(--color-border); border-radius: 3px; overflow: hidden; margin-top: 8px; }
.usage-fill { height: 100%; background: var(--color-primary); border-radius: 3px; transition: width 0.3s ease; }
.usage-fill.usage-warning { background: var(--color-warning); }
.file-input { display: none; }

/* PAGE LIST */
.page-list { width: 100%; }
.page-list-title {
  font-size: 0.95rem; font-weight: 600; color: var(--color-text-secondary);
  margin-bottom: 10px; text-transform: uppercase; letter-spacing: 0.05em;
}
.page-items { display: flex; flex-wrap: wrap; gap: 8px; }
.page-item {
  position: relative; width: 80px; height: 100px;
  border: 2px solid var(--color-border); border-radius: var(--radius-sm);
  overflow: hidden; background: var(--color-surface-solid); cursor: pointer;
  transition: var(--transition);
}
.page-item:hover { border-color: var(--color-primary); box-shadow: var(--shadow-sm); }
.page-thumb { width: 100%; height: 70px; object-fit: cover; }
.page-number {
  display: block; text-align: center; font-size: 0.75rem; font-weight: 600;
  color: var(--color-text-secondary); padding: 2px 0;
}
.page-remove {
  position: absolute; top: 4px; right: 4px;
  width: 22px; height: 22px; border-radius: 50%;
  background: rgba(239, 68, 68, 0.9); border: none;
  color: white; cursor: pointer; display: flex;
  align-items: center; justify-content: center;
  transition: var(--transition);
}
.page-remove:hover { background: #dc2626; transform: scale(1.1); }

/* FINALIZE */
.finalize-section { width: 100%; }
.btn-full { width: 100%; }
.btn-success {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white; box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
}
.btn-success:hover { transform: translateY(-1px); box-shadow: 0 6px 20px rgba(16, 185, 129, 0.4); }

/* CHOOSE */
.choose-actions { display: flex; gap: 16px; width: 100%; }
.choose-btn {
  flex: 1; display: flex; flex-direction: column; align-items: center; gap: 14px;
  padding: 32px 20px; border: 2px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-surface-solid); cursor: pointer; transition: var(--transition);
}
.choose-btn:hover {
  border-color: var(--color-primary);
  background: var(--color-surface-accent);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}
.choose-icon { font-size: 2.5rem; }
.choose-icon-wrap {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.camera-icon-wrap { background: linear-gradient(135deg, #6366f1, #8b5cf6); box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3); }
.file-icon-wrap { background: linear-gradient(135deg, #f59e0b, #ef4444); box-shadow: 0 4px 12px rgba(245, 158, 11, 0.3); }
.choose-label { font-size: 0.95rem; font-weight: 600; color: var(--color-text); }

/* CAMERA */
.camera-container {
  position: relative; width: 100%; max-height: 60vh;
  overflow: hidden; border-radius: var(--radius-md); background: #000;
  box-shadow: var(--shadow-lg);
}
.camera-video { width: 100%; display: block; object-fit: cover; max-height: 60vh; border: 3px solid transparent; transition: border-color 0.4s ease; }
.camera-video.tilt-good { border-color: #34c759; animation: tilt-pulse 2s ease-in-out infinite; }
.camera-video.tilt-ok { border-color: #f5b301; }
.camera-video.tilt-bad { border-color: #ef4444; }
.camera-video.tilt-filled { border-color: #34c759; }

/* Tilt indicator HUD */
.angle-hud {
  position: absolute; top: 10px; left: 50%; transform: translateX(-50%);
  display: flex; align-items: center; gap: 8px;
  padding: 6px 14px; border-radius: 999px;
  background: rgba(0, 0, 0, 0.55); backdrop-filter: blur(4px);
  color: white; font-size: 0.85rem; font-weight: 500;
  transition: opacity 0.3s ease;
}
.angle-dot {
  width: 10px; height: 10px; border-radius: 50%;
  transition: background 0.4s ease;
}
.dot-good { background: #34c759; box-shadow: 0 0 8px #34c759; }
.dot-ok { background: #f5b301; box-shadow: 0 0 8px #f5b301; }
.dot-bad { background: #ef4444; box-shadow: 0 0 8px #ef4444; }
.dot-filled { background: #34c759; box-shadow: 0 0 8px #34c759; }

@keyframes tilt-pulse {
  0%, 100% { border-color: #34c759; }
  50% { border-color: rgba(52, 199, 89, 0.35); }
}
.camera-canvas-hidden { display: none; }
.camera-loading {
  position: absolute; top: 0; left: 0; width: 100%; height: 100%;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 12px; color: white; background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
}
.camera-loading p { font-weight: 500; }
.camera-controls { display: flex; align-items: center; justify-content: space-between; width: 100%; }
.capture-btn {
  width: 68px; height: 68px; border-radius: 50%; border: 4px solid white;
  background: var(--color-primary); cursor: pointer; display: flex; align-items: center; justify-content: center;
  transition: var(--transition); box-shadow: 0 4px 16px rgba(99, 102, 241, 0.4);
}
.capture-btn:active { transform: scale(0.9); }
.capture-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.capture-ring { width: 52px; height: 52px; border-radius: 50%; border: 3px solid white; }

/* EDIT */
.preview-section { width: 100%; text-align: center; overflow: hidden; }
.preview-image-container {
  max-width: 100%; max-height: 45vh; overflow: hidden;
  border: 2px solid var(--color-border); border-radius: var(--radius-md);
  margin-bottom: 10px; transition: transform 0.3s;
  box-shadow: var(--shadow-sm);
}
.preview-image { width: 100%; height: auto; object-fit: contain; max-height: 45vh; }
.file-info { color: var(--color-text-muted); font-size: 0.8rem; word-break: break-all; font-family: 'SF Mono', 'Fira Code', monospace; }

.edit-toolbar { display: flex; gap: 10px; }
.toolbar-btn {
  width: 50px; height: 50px; border: 2px solid var(--color-border); border-radius: 14px;
  background: white; font-size: 1.4rem; cursor: pointer; transition: var(--transition);
  display: flex; align-items: center; justify-content: center;
}
.toolbar-btn:hover {
  border-color: var(--color-primary); background: var(--color-primary-light);
  color: var(--color-primary);
}

/* CROP */
.crop-container { width: 100%; touch-action: none; }
.crop-canvas { width: 100%; border-radius: var(--radius-md); cursor: crosshair; box-shadow: var(--shadow-sm); }

/* COMMON */
.error-message { color: var(--color-error); font-size: 0.85rem; font-weight: 500; text-align: center; }
.success-message { color: var(--color-success); font-size: 0.85rem; font-weight: 500; text-align: center; }
.actions { display: flex; gap: 12px; width: 100%; }
.btn {
  flex: 1; padding: 12px 20px; border: none; border-radius: var(--radius-sm);
  cursor: pointer; font-size: 0.95rem; font-weight: 600; font-family: inherit;
  transition: var(--transition);
}
.btn-primary {
  background: var(--color-primary); color: white;
  box-shadow: 0 4px 14px rgba(99, 102, 241, 0.35);
}
.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover); transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
}
.btn-primary:disabled { background: var(--color-text-muted); box-shadow: none; cursor: not-allowed; }
.btn-secondary {
  background: transparent; color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}
.btn-secondary:hover:not(:disabled) { background: var(--color-surface-muted); }

.spinner {
  width: 36px; height: 36px; border: 3px solid var(--color-border);
  border-top-color: var(--color-primary); border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
