import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock WebSocket globally
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  static instances: MockWebSocket[] = []
  url: string
  readyState = 0 // CONNECTING
  onopen: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  onmessage: ((ev: MessageEvent) => void) | null = null
  onerror: ((ev: Event) => void) | null = null
  sent: string[] = []

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  simulateOpen() {
    this.readyState = MockWebSocket.OPEN
    if (this.onopen) this.onopen(new Event('open'))
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code: 1000 }))
    }
  }

  simulateMessage(data: string) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data }))
    }
  }
}

// Mock apiService
vi.mock('../utils/api', () => ({
  apiService: {
    getWebSocketURL: (sessionID: string) => `wss://test.example.com/ws/session/${sessionID}`,
  },
}))

describe('WebSocketClient', () => {
  let websocketClient: typeof import('../utils/websocket').websocketClient

  beforeEach(async () => {
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
    vi.useFakeTimers()

    const pinia = createPinia()
    setActivePinia(pinia)

    vi.resetModules()
    const mod = await import('../utils/websocket')
    websocketClient = mod.websocketClient
  })

  afterEach(() => {
    websocketClient.disconnect()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  describe('on / off', () => {
    it('should register and call handler when event is received', () => {
      const handler = vi.fn()
      websocketClient.on('image_uploaded', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(JSON.stringify({ event: 'image_uploaded', data: {} }))

      expect(handler).toHaveBeenCalledOnce()
      expect(handler).toHaveBeenCalledWith({})
    })

    it('should NOT call handler twice for a single message', () => {
      const handler = vi.fn()
      websocketClient.on('download_confirmed', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(JSON.stringify({ event: 'download_confirmed', data: { session_id: 'abc' } }))

      expect(handler).toHaveBeenCalledOnce()
    })

    it('should remove handler with off()', () => {
      const handler = vi.fn()
      websocketClient.on('image_uploaded', handler)
      websocketClient.off('image_uploaded', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(JSON.stringify({ event: 'image_uploaded', data: {} }))

      expect(handler).not.toHaveBeenCalled()
    })

    it('should only remove the specified handler, not others', () => {
      const handler1 = vi.fn()
      const handler2 = vi.fn()
      websocketClient.on('image_uploaded', handler1)
      websocketClient.on('image_uploaded', handler2)
      websocketClient.off('image_uploaded', handler1)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(JSON.stringify({ event: 'image_uploaded', data: {} }))

      expect(handler1).not.toHaveBeenCalled()
      expect(handler2).toHaveBeenCalledOnce()
    })

    it('should support multiple handlers for the same event', () => {
      const handler1 = vi.fn()
      const handler2 = vi.fn()
      websocketClient.on('image_uploaded', handler1)
      websocketClient.on('image_uploaded', handler2)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(JSON.stringify({ event: 'image_uploaded', data: { foo: 'bar' } }))

      expect(handler1).toHaveBeenCalledOnce()
      expect(handler2).toHaveBeenCalledOnce()
    })

    it('should not call handlers for different event types', () => {
      const handler = vi.fn()
      websocketClient.on('image_uploaded', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(JSON.stringify({ event: 'download_request', data: {} }))

      expect(handler).not.toHaveBeenCalled()
    })

    it('should handle off for non-existent handler gracefully', () => {
      const handler = vi.fn()
      // off on non-existent event should not throw
      expect(() => websocketClient.off('error', handler)).not.toThrow()
    })
  })

  describe('send', () => {
    it('should send JSON message when connected', () => {
      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()

      websocketClient.send('download_request', { session_id: 'abc' })

      const ws = MockWebSocket.instances[0]!
      expect(ws.sent).toHaveLength(1)
      const sent = JSON.parse(ws.sent[0]!)
      expect(sent.event).toBe('download_request')
      expect(sent.data).toEqual({ session_id: 'abc' })
    })

    it('should not send when not connected', () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      websocketClient.send('download_request', { session_id: 'abc' })
      expect(consoleSpy).toHaveBeenCalledWith('WebSocket is not connected, cannot send message')
    })
  })

  describe('connect / disconnect', () => {
    it('should create a WebSocket connection on connect', () => {
      websocketClient.connect('test-session-id')
      expect(MockWebSocket.instances).toHaveLength(1)
      expect(MockWebSocket.instances[0]!.url).toBe('wss://test.example.com/ws/session/test-session-id')
    })

    it('should close existing connection before reconnecting', () => {
      websocketClient.connect('session-1')
      MockWebSocket.instances[0]!.simulateOpen()
      const firstWs = MockWebSocket.instances[0]!

      websocketClient.connect('session-2')
      expect(firstWs.readyState).toBe(3) // CLOSED
      expect(MockWebSocket.instances).toHaveLength(2)
    })

    it('should set socket to null on disconnect', () => {
      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      expect(websocketClient.isConnected()).toBe(true)

      websocketClient.disconnect()
      expect(websocketClient.isConnected()).toBe(false)
    })

    it('should return correct ready state', () => {
      expect(websocketClient.getReadyState()).toBeNull()

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      expect(websocketClient.getReadyState()).toBe(1)
      expect(websocketClient.isConnected()).toBe(true)
    })
  })

  describe('reconnection', () => {
    it('should attempt to reconnect after unexpected close', async () => {
      // Set sessionID so reconnect logic works
      const { useSessionStore } = await import('../stores/sessionStore')
      const store = useSessionStore()
      store.setSessionID('test-session-id')

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()

      // Simulate unexpected close
      const ws = MockWebSocket.instances[0]!
      ws.readyState = 3
      if (ws.onclose) {
        ws.onclose(new CloseEvent('close', { code: 1006 }))
      }

      // Advance timer to trigger reconnection
      vi.advanceTimersByTime(3000)

      expect(MockWebSocket.instances).toHaveLength(2)
    })

    it('should emit status_update disconnected on close', () => {
      const handler = vi.fn()
      websocketClient.on('status_update', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()

      MockWebSocket.instances[0]!.close()

      expect(handler).toHaveBeenCalledWith({ status: 'disconnected' })
    })
  })
})
