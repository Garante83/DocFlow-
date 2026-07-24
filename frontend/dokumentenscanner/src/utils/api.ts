// API Utility for Dokumentenscanner
// Handles all HTTP requests to the backend

import axios from 'axios'
import { useSessionStore } from '../stores/sessionStore'

// Create axios instance with base configuration
// When running standalone, use the configured backend URL
// When embedded in backend, use relative paths (baseURL will be empty/undefined)
const api = axios.create({
  baseURL: import.meta.env.VITE_BACKEND_URL || '',
  timeout: 30000,
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
    if (error.code === 'ECONNREFUSED') {
      console.error('Connection refused - Backend server is not running')
    } else if (error.response) {
      const status = error.response.status
      
      switch (status) {
        case 401:
          console.error('Unauthorized - Invalid PIN or session')
          break
        case 403:
          console.error('Forbidden - Access denied')
          break
        case 404:
          console.error('Not found - Endpoint does not exist')
          break
        case 429:
          console.error('Too many requests - Rate limited')
          break
        case 500:
          console.error('Internal server error')
          break
        default:
          console.error(`Error ${status}: ${error.response.data}`)
      }
    } else if (error.request) {
      console.error('No response received from server')
    }
    
    return Promise.reject(error)
  }
)

// API Endpoints
export interface CreateSessionResponse {
  session_id: string
  pin: string
  message: string
}

export interface VerifyPINResponse {
  valid: boolean
  message: string
  session_id?: string
}

export interface UploadImageResponse {
  image_id: string
  message: string
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
      pdf_url: response.data.pdf_url
    }
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

  // WebSocket URL
  getWebSocketURL(sessionID: string): string {
    const backendURL = import.meta.env.VITE_BACKEND_URL || window.location.origin
    const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const host = backendURL.replace(/^https?:\/\//, '') || window.location.host
    return `${wsProtocol}://${host}/ws/session/${sessionID}`
  },
}

export default api
