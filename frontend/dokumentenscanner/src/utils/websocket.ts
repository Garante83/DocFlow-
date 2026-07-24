// WebSocket Utility for Dokumentenscanner
// Manages real-time communication with the backend

import { useSessionStore } from '../stores/sessionStore'
import { apiService } from './api'

export type WebSocketEventType = 
  | 'session_created'
  | 'pin_verified'
  | 'image_uploaded'
  | 'pdf_ready'
  | 'session_locked'
  | 'error'
  | 'status_update'
  | 'download_request'
  | 'download_confirmed'

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

  connect(sessionID: string): void {
    // Close existing connection if any
    this.disconnect()

    const wsURL = apiService.getWebSocketURL(sessionID)
    
    try {
      this.socket = new WebSocket(wsURL)
      
      this.socket.onopen = () => {
        this.reconnectAttempts = 0
        console.log('WebSocket connection established')
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
        
        // Attempt to reconnect
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          setTimeout(() => {
            if (this.sessionStore.sessionID) {
              this.reconnectAttempts++
              this.connect(this.sessionStore.sessionID)
            }
          }, this.reconnectTimeout)
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
      this.socket.close()
      this.socket = null
    }
    this.reconnectAttempts = 0
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

    // Also emit a generic event
    this.emitEvent(message.event, message.data)
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
