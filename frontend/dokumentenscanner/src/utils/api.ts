// API Utility for DocFlow
// Handles all HTTP requests to the backend

import axios from 'axios'
import { useSessionStore } from '../stores/sessionStore'
import i18n from '../i18n'

const t = i18n.global.t

// Create axios instance with base configuration
// When running standalone, use the configured backend URL
// When embedded in backend, use relative paths (baseURL will be empty/undefined)
const api = axios.create({
  baseURL: import.meta.env.VITE_BACKEND_URL || '',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add session ID and authorization
api.interceptors.request.use(
  (config) => {
    const sessionStore = useSessionStore()
    
    if (sessionStore.sessionID) {
      config.headers.Authorization = `Bearer ${sessionStore.sessionID}`
    }
    
    // Add PIN if available and not already set
    if (sessionStore.pin && !config.headers.pin) {
      config.headers.pin = sessionStore.pin
    }
    
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor to handle errors globally
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Extract meaningful error message from backend response
    let message = t('api.unexpectedError')
    
    if (error.code === 'ECONNREFUSED') {
      message = t('api.connectionRefused')
    } else if (error.response) {
      const status = error.response.status
      const data = error.response.data
      
      // Use backend error message if available
      if (data && typeof data === 'object' && data.error) {
        message = data.error
      } else {
        switch (status) {
          case 400:
            message = t('api.invalidRequest')
            break
          case 401:
            message = t('api.invalidPin')
            break
          case 403:
            message = t('api.sessionNotReady')
            break
          case 404:
            message = t('api.sessionNotFound')
            break
          case 413:
            message = t('api.fileTooLarge')
            break
          case 429:
            message = t('api.tooManyRequests')
            break
          case 500:
            message = t('api.serverError')
            break
          default:
            message = `Error ${status}: ${data?.message || t('api.unexpectedError')}`
        }
      }
    } else if (error.request) {
      message = t('api.noResponse')
    }
    
    // Attach message to error for callers to use
    error.userMessage = message
    return Promise.reject(error)
  }
)

// API Endpoints
export interface CreateSessionResponse {
  session_id: string
  pin: string
  message?: string
  max_file_size_mb?: number
  max_pages?: number
}

export interface VerifyPINResponse {
  valid: boolean
  message: string
  session_id?: string
}

export interface UploadImageResponse {
  image_id: string
  message: string
  page_count?: number
  pdf_url?: string
}

export interface PDFStatusResponse {
  status: string
  pdf_url?: string
  message?: string
}

export interface LockStatusResponse {
  locked: boolean
  locked_until?: string
  attempts_remaining?: number
}

// API Functions - Adapted to match backend routes
export const apiService = {
  // Session management
  async createSession(): Promise<CreateSessionResponse> {
    const response = await api.post<CreateSessionResponse>('/api/session')
    return {
      session_id: response.data.session_id,
      pin: response.data.pin,
      message: 'Session created'
    }
  },

  async verifyPIN(sessionID: string, pin: string): Promise<VerifyPINResponse> {
    const response = await api.post<VerifyPINResponse>(`/api/session/${sessionID}/verify-pin`, {
      pin: pin,
    })
    return {
      valid: response.data.valid || response.status === 200,
      message: response.data.message || (response.data.valid ? 'PIN verified' : 'Invalid PIN'),
      session_id: sessionID
    }
  },

  async getLockStatus(): Promise<LockStatusResponse> {
    // Backend doesn't have explicit lock status endpoint
    // For now, return default values
    return {
      locked: false,
      attempts_remaining: 3
    }
  },

  // Image upload
  async uploadImage(sessionID: string, image: File): Promise<UploadImageResponse> {
    const formData = new FormData()
    formData.append('image', image)
    
    const response = await api.post<UploadImageResponse>(`/api/session/${sessionID}/upload`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    return {
      image_id: sessionID,
      message: response.data.message || 'Image uploaded successfully',
      page_count: response.data.page_count,
      pdf_url: response.data.pdf_url
    }
  },

  // Finalize upload and generate PDF
  async finalizeUpload(sessionID: string): Promise<void> {
    await api.post(`/api/session/${sessionID}/finalize`)
  },

  // PDF generation and retrieval
  async getPDFStatus(): Promise<PDFStatusResponse> {
    // Backend doesn't have explicit status endpoint
    // Assume PDF is ready if we can access it
    return {
      status: 'ready',
      message: 'PDF ready for download'
    }
  },

  async downloadPDF(sessionID: string): Promise<Blob> {
    const response = await api.get(`/api/session/${sessionID}/pdf`, {
      responseType: 'blob',
    })
    return response.data
  },

  // WebSocket URL with authentication token
  getWebSocketURL(sessionID: string, token: string): string {
    const backendURL = import.meta.env.VITE_BACKEND_URL || window.location.origin
    const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const host = backendURL.replace(/^https?:\/\//, '') || window.location.host
    return `${wsProtocol}://${host}/ws/session/${sessionID}?token=${encodeURIComponent(token)}`
  },
}

export default api
