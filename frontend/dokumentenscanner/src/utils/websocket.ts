// WebSocket Utility for Dokumentenscanner
// Manages real-time communication with the backend

import { useSessionStore } from '../stores/sessionStore'
import { apiService } from './api'

export type WebSocketEventType = 
  | 'session_created'
  | 'pin_verified'
  | 'image_uploaded'
  | 'image_added'
  | 'pdf_ready'
  | 'session_locked'
  | 'error'
  | 'status_update'
  | 'download_request'
  | 'download_confirmed'
  | 'finalize_upload'
  | 'page_removed'

export interface WebSocketEvent {
  type: WebSocketEventType
  data: unknown
  timestamp: string
}

export interface WebSocketMessage {
  event: WebSocketEventType
  data: unknown
}

class WebSocketClient {
  private socket: WebSocket | null = null
  private reconnectAttempts: number = 0
  private maxReconnectAttempts: number = 5
  private reconnectTimeout: number = 3000
  private eventHandlers: Map<WebSocketEventType, ((data: unknown) => void)[]> = new Map()
  private sessionStore = useSessionStore()
  private authToken: string = ''
  private closingSocket: WebSocket | null = null
  private reconnectTimer: number | null = null

  connect(sessionID: string, token?: string): void {
    // Close existing connection if any (without triggering a reconnect)
    this.disconnect()
    this.authToken = token || this.sessionStore.pin || ''

    const wsURL = apiService.getWebSocketURL(sessionID)

    try {
      this.socket = new WebSocket(wsURL)

      this.socket.onopen = () => {
        this.reconnectAttempts = 0
        console.log('WebSocket connection established')
        // Authenticate immediately with the first message (never via URL query)
        this.socket?.send(JSON.stringify({ type: 'auth', token: this.authToken }))
        this.emitEvent('status_update', { status: 'connected' })
      }

      this.socket.onmessage = (event: MessageEvent) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          this.handleMessage(message)
        } catch (error) {
          console.error('Error parsing WebSocket message:', error)
        }
      }

      this.socket.onclose = (event: CloseEvent) => {
        console.log(`WebSocket connection closed: code=${event.code}, reason=${event.reason}`)
        this.emitEvent('status_update', { status: 'disconnected' })

        // Deliberate disconnects (via disconnect()) must never trigger a
        // reconnect. Compare by socket reference: the close event of a
        // replaced socket can arrive asynchronously after a new connection
        // has already been opened.
        if (this.closingSocket !== null && event.target === this.closingSocket) {
          this.closingSocket = null
          return
        }

        // Reconnect with exponential backoff on real connection losses
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          const delay = this.reconnectTimeout * Math.pow(2, this.reconnectAttempts)
          this.reconnectTimer = window.setTimeout(() => {
            this.reconnectTimer = null
            if (this.sessionStore.sessionID) {
              this.reconnectAttempts++
              this.connect(this.sessionStore.sessionID)
            }
          }, delay)
        }
      }

      this.socket.onerror = (error: Event) => {
        console.error('WebSocket error:', error)
        this.emitEvent('error', { message: 'WebSocket error' })
      }
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error)
      this.emitEvent('error', { message: 'Failed to create WebSocket connection' })
    }
  }

  disconnect(): void {
    if (this.socket) {
      this.closingSocket = this.socket
      this.socket.close()
      this.socket = null
    }
    // reconnectAttempts is deliberately NOT reset here: connect() goes
    // through disconnect(), and resetting here would zero the backoff on
    // every reconnect. onopen resets it once a connection is established.
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  on(event: WebSocketEventType, handler: (data: unknown) => void): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, [])
    }
    this.eventHandlers.get(event)!.push(handler)
  }

  off(event: WebSocketEventType, handler: (data: unknown) => void): void {
    const handlers = this.eventHandlers.get(event)
    if (handlers) {
      const index = handlers.indexOf(handler)
      if (index !== -1) {
        handlers.splice(index, 1)
      }
    }
  }

  send(event: WebSocketEventType, data: unknown): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      const message: WebSocketMessage = { event, data }
      this.socket.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket is not connected, cannot send message')
    }
  }

  private handleMessage(message: WebSocketMessage): void {
    console.log('Received WebSocket message:', message.event, message.data)
    
    // Emit to all handlers for this event type
    const handlers = this.eventHandlers.get(message.event)
    if (handlers) {
      handlers.forEach(handler => handler(message.data))
    }
  }

  private emitEvent(event: WebSocketEventType, data: unknown): void {
    const handlers = this.eventHandlers.get(event)
    if (handlers) {
      handlers.forEach(handler => handler(data))
    }
  }

  isConnected(): boolean {
    return this.socket !== null && this.socket.readyState === WebSocket.OPEN
  }

  getReadyState(): number | null {
    return this.socket?.readyState ?? null
  }
}

// Singleton instance
export const websocketClient = new WebSocketClient()

export default websocketClient
