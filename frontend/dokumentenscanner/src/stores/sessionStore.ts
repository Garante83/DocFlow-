// Session Store for Dokumentenscanner
// Manages session state, PIN verification, and multi-page image upload status

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

/**
 * Session Status Enum
 */
export type SessionStatus = 'waiting_for_pin' | 'upload_allowed' | 'uploading' | 'uploaded' | 'ready' | 'downloaded'

/**
 * Session Store - Central state management for the scanning session
 */
export const useSessionStore = defineStore('session', () => {
  // State
  const sessionID = ref<string | null>(null)
  const status = ref<SessionStatus>('waiting_for_pin')
  const images = ref<File[]>([])
  const pdfData = ref<string | Blob | null>(null)
  const pin = ref<string>('')
  const failedAttempts = ref<number>(0)
  const lockedUntil = ref<Date | null>(null)
  const frontendURL = ref<string>(import.meta.env.VITE_FRONTEND_URL || 'http://localhost:8080')
  const backendURL = ref<string>(import.meta.env.VITE_BACKEND_URL || window.location.origin)
  const maxFileSizeMB = ref<number>(10)
  const maxPages = ref<number>(20)

  // Getters
  const isActive = computed(() => !!sessionID.value)
  const isLocked = computed(() => {
    if (lockedUntil.value === null) return false
    return new Date() < lockedUntil.value
  })
  const apiBaseURL = computed(() => backendURL.value)
  const imageCount = computed(() => images.value.length)
  const hasImages = computed(() => images.value.length > 0)

  // Actions
  function setSessionID(id: string) {
    sessionID.value = id
  }

  function setStatus(newStatus: SessionStatus) {
    status.value = newStatus
  }

  function addImage(image: File) {
    images.value.push(image)
  }

  function removeImage(index: number) {
    if (index >= 0 && index < images.value.length) {
      images.value.splice(index, 1)
    }
  }

  function reorderImage(from: number, to: number) {
    if (from < 0 || from >= images.value.length || to < 0 || to >= images.value.length) return
    const [item] = images.value.splice(from, 1)
    images.value.splice(to, 0, item)
  }

  function setPDF(pdf: string | Blob) {
    pdfData.value = pdf
  }

  function setPIN(newPin: string) {
    pin.value = newPin
  }

  function setLimits(size: number, pages: number) {
    maxFileSizeMB.value = size
    maxPages.value = pages
  }

  function incrementFailedAttempts() {
    failedAttempts.value++
  }

  function resetFailedAttempts() {
    failedAttempts.value = 0
  }

  function setLockedUntil(timestamp: Date | string | null) {
    if (timestamp === null) {
      lockedUntil.value = null
    } else if (typeof timestamp === 'string') {
      lockedUntil.value = new Date(timestamp)
    } else {
      lockedUntil.value = timestamp
    }
  }

  function reset() {
    sessionID.value = null
    status.value = 'waiting_for_pin'
    images.value = []
    pdfData.value = null
    pin.value = ''
    failedAttempts.value = 0
    lockedUntil.value = null
    maxFileSizeMB.value = 10
    maxPages.value = 20
  }

  return {
    sessionID,
    status,
    images,
    pdfData,
    pin,
    failedAttempts,
    lockedUntil,
    frontendURL,
    backendURL,
    maxFileSizeMB,
    maxPages,
    isActive,
    isLocked,
    apiBaseURL,
    imageCount,
    hasImages,
    setSessionID,
    setStatus,
    addImage,
    removeImage,
    reorderImage,
    setPDF,
    setPIN,
    setLimits,
    incrementFailedAttempts,
    resetFailedAttempts,
    setLockedUntil,
    reset,
  }
})
