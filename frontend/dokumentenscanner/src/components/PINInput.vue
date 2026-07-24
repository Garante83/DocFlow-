<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useSessionStore } from '../stores/sessionStore'
import { apiService } from '../utils/api'
import { websocketClient } from '../utils/websocket'

interface Emits {
  (e: 'verified'): void
  (e: 'back'): void
}

const emit = defineEmits<Emits>()
const sessionStore = useSessionStore()

const pinInput = ref('')
const isLoading = ref(false)
const errorMessage = ref<string | null>(null)
const showBackButton = computed(() => sessionStore.failedAttempts > 0)
const remainingAttempts = computed(() => {
  return Math.max(0, 3 - sessionStore.failedAttempts)
})

const isLocked = computed(() => {
  if (sessionStore.lockedUntil === null) return false
  return new Date() < sessionStore.lockedUntil
})

const lockCountdown = ref('')

const verifyPIN = async () => {
  if (pinInput.value.length < 6) return
  await attemptVerification()
}

const attemptVerification = async () => {
  if (isLocked.value) {
    errorMessage.value = `Account locked. Please wait ${lockCountdown.value} before trying again.`
    return
  }

  if (pinInput.value.length !== 6) {
    errorMessage.value = 'PIN must be 6 digits'
    return
  }

  if (!sessionStore.sessionID) {
    errorMessage.value = 'No session ID. Please refresh the page.'
    return
  }

  isLoading.value = true
  errorMessage.value = null

  try {
    const response = await apiService.verifyPIN(sessionStore.sessionID, pinInput.value)

    if (response.valid) {
      sessionStore.setPIN(pinInput.value)
      sessionStore.resetFailedAttempts()
      sessionStore.setSessionID(response.session_id || sessionStore.sessionID)
      sessionStore.setStatus('upload_allowed')

      websocketClient.connect(sessionStore.sessionID)

      emit('verified')
    } else {
      sessionStore.incrementFailedAttempts()
      errorMessage.value = response.message || 'Invalid PIN'
      pinInput.value = ''

      if (sessionStore.failedAttempts >= 3) {
        sessionStore.setLockedUntil(new Date(Date.now() + 30000))
        errorMessage.value = 'Too many failed attempts. Account locked for 30 seconds.'
      }
    }
  } catch (error) {
    errorMessage.value = 'Connection error. Please check your network.'
    console.error('Verification error:', error)
  } finally {
    isLoading.value = false
  }
}

const updateLockCountdown = () => {
  if (isLocked.value && sessionStore.lockedUntil) {
    const now = new Date()
    const diff = sessionStore.lockedUntil.getTime() - now.getTime()

    if (diff <= 0) {
      sessionStore.setLockedUntil(null)
      sessionStore.resetFailedAttempts()
      lockCountdown.value = ''
    } else {
      const seconds = Math.ceil(diff / 1000)
      lockCountdown.value = `${seconds}s`
    }
  }
}

let countdownInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  countdownInterval = setInterval(updateLockCountdown, 1000)
})

onUnmounted(() => {
  if (countdownInterval) clearInterval(countdownInterval)
})

function handleBack() {
  emit('back')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && pinInput.value.length === 6) {
    verifyPIN()
  }
}

function handlePaste(e: ClipboardEvent) {
  const pasted = e.clipboardData?.getData('text') || ''
  const digits = pasted.replace(/\D/g, '').slice(0, 6)
  if (digits.length === 6) {
    pinInput.value = digits
    setTimeout(() => verifyPIN(), 100)
  }
}
</script>

<template>
  <div class="pin-input-container" @keydown="handleKeydown">
    <div class="pin-header">
      <div class="pin-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
        </svg>
      </div>
      <h2>Enter PIN</h2>
      <p class="info">Enter the 6-digit PIN to start scanning</p>
    </div>

    <div class="input-group">
      <input
        v-model="pinInput"
        type="text"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="6"
        @input="errorMessage = null"
        @paste="handlePaste"
        @keydown="handleKeydown"
        :disabled="isLoading || isLocked"
        placeholder="\u2022\u2022\u2022\u2022\u2022\u2022"
        class="pin-input"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
      />

      <div v-if="isLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>
    </div>

    <div v-if="errorMessage" class="error-message">
      {{ errorMessage }}
    </div>

    <div v-if="showBackButton && !isLocked" class="attempts-info">
      <span v-if="remainingAttempts > 0">
        {{ remainingAttempts }} attempt{{ remainingAttempts !== 1 ? 's' : '' }} remaining
      </span>
    </div>

    <div v-if="isLocked" class="locked-message">
      <p>Account locked for security</p>
      <p class="countdown">{{ lockCountdown }}</p>
    </div>

    <div class="actions">
      <button
        @click="handleBack"
        class="btn btn-ghost"
        :disabled="isLoading"
        v-if="showBackButton || sessionStore.sessionID"
      >
        Start Over
      </button>

      <button
        @click="verifyPIN"
        class="btn btn-primary"
        :disabled="isLoading || isLocked || pinInput.length !== 6"
      >
        <span v-if="isLoading">Verifying...</span>
        <span v-else>Verify PIN</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.pin-input-container {
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

.pin-header {
  text-align: center;
}

.pin-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 12px;
  background: var(--color-primary-bg);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 6px 18px rgba(99, 102, 241, 0.3);
}

.pin-header h2 {
  font-size: 1.4rem;
  font-weight: 700;
  color: var(--color-text);
  margin-bottom: 4px;
}

.info {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.input-group {
  position: relative;
  width: 100%;
}

.pin-input {
  width: 100%;
  padding: 16px 20px;
  font-size: 1.8rem;
  text-align: center;
  letter-spacing: 0.4em;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  outline: none;
  transition: var(--transition);
  background: white;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-weight: 700;
  color: var(--color-text);
}

.pin-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.1);
}

.pin-input:disabled {
  background-color: #f8fafc;
  cursor: not-allowed;
  color: var(--color-text-muted);
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(4px);
  border-radius: var(--radius-md);
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

.error-message {
  color: var(--color-error);
  font-size: 0.85rem;
  font-weight: 500;
  text-align: center;
  min-height: 20px;
}

.attempts-info {
  color: var(--color-warning);
  font-size: 0.8rem;
  font-weight: 500;
  text-align: center;
}

.locked-message {
  color: var(--color-error);
  text-align: center;
}

.countdown {
  font-weight: 700;
  margin-top: 4px;
  font-size: 1.2rem;
}

.actions {
  display: flex;
  gap: 12px;
  width: 100%;
}

.btn {
  flex: 1;
  padding: 12px 20px;
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

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
}

.btn-primary:disabled {
  background: #94a3b8;
  box-shadow: none;
  cursor: not-allowed;
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.btn-ghost:hover:not(:disabled) {
  background: #f1f5f9;
}
</style>
