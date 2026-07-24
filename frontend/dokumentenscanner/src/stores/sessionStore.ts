// Session Store for Dokumentenscanner
// Manages session state, PIN verification, and image upload status

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

/**
 * Session Status Enum
 */
export type SessionStatus = 'waiting_for_pin' | 'upload_allowed' | 'uploaded' | 'ready' | 'downloaded'

/**
 * Session Store - Central state management for the scanning session
 */
export const useSessionStore = defineStore('session', () => {
  // State
  const sessionID = ref<string | null>(null)
  const status = ref<SessionStatus>('waiting_for_pin')
  const imageData = ref<string | File | null>(null)
  const pdfData = ref<string | Blob | null>(null)
  const pin = ref<string>('')
  const failedAttempts = ref<number>(0)
  const lockedUntil = ref<Date | null>(null)
  const frontendURL = ref<string>(import.meta.env.VITE_FRONTEND_URL || 'http://localhost:8080')
  const backendURL = ref<string>(import.meta.env.VITE_BACKEND_URL || window.location.origin)

  // Getters
  const isActive = computed(() => !!sessionID.value)
  const isLocked = computed(() => {
    if (lockedUntil.value === null) return false
    return new Date() < lockedUntil.value
  })
  const apiBaseURL = computed(() => backendURL.value)

  // Actions
  function setSessionID(id: string) {
    sessionID.value = id
  }

  function setStatus(newStatus: SessionStatus) {
    status.value = newStatus
  }

  function setImage(image: string | File) {
    imageData.value = image
  }

  function setPDF(pdf: string | Blob) {
    pdfData.value = pdf
  }

  function setPIN(newPin: string) {
    pin.value = newPin
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
    imageData.value = null
    pdfData.value = null
    pin.value = ''
    failedAttempts.value = 0
    lockedUntil.value = null
  }

  return {
    sessionID,
    status,
    imageData,
    pdfData,
    pin,
    failedAttempts,
    lockedUntil,
    frontendURL,
    backendURL,
    isActive,
    isLocked,
    apiBaseURL,
    setSessionID,
    setStatus,
    setImage,
    setPDF,
    setPIN,
    incrementFailedAttempts,
    resetFailedAttempts,
    setLockedUntil,
    reset,
  }
})
