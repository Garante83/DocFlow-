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

      expect(handler).toHaveBeenCalledExactlyOnceWith({})
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

    it('should pass flat backend messages (no data object) as payload to handlers', () => {
      const handler = vi.fn()
      websocketClient.on('image_added', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(
        JSON.stringify({ event: 'image_added', session_id: 'abc', page_count: 2 })
      )

      expect(handler).toHaveBeenCalledExactlyOnceWith({
        session_id: 'abc',
        page_count: 2,
      })
    })

    it('should pass nested data payloads unchanged to handlers', () => {
      const handler = vi.fn()
      websocketClient.on('download_request', handler)

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()
      MockWebSocket.instances[0]!.simulateMessage(
        JSON.stringify({ event: 'download_request', data: { session_id: 'abc' } })
      )

      expect(handler).toHaveBeenCalledExactlyOnceWith({ session_id: 'abc' })
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
    it('should send auth message on open and JSON message when connected', () => {
      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()

      websocketClient.send('download_request', { session_id: 'abc' })

      const ws = MockWebSocket.instances[0]!
      // First message is the auth handshake (never in the URL), then the actual message
      expect(ws.sent).toHaveLength(2)
      const auth = JSON.parse(ws.sent[0]!)
      expect(auth.type).toBe('auth')
      expect(JSON.stringify(auth)).not.toContain('download_request')
      const sent = JSON.parse(ws.sent[1]!)
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

    it('should NOT reconnect after deliberate disconnect()', async () => {
      const { useSessionStore } = await import('../stores/sessionStore')
      const store = useSessionStore()
      store.setSessionID('test-session-id')

      websocketClient.connect('test-session-id')
      MockWebSocket.instances[0]!.simulateOpen()

      websocketClient.disconnect()

      // No reconnect may fire - neither immediately nor later
      vi.advanceTimersByTime(60000)
      expect(MockWebSocket.instances).toHaveLength(1)
    })

    it('should NOT let the async close of a replaced socket trigger a reconnect', async () => {
      const { useSessionStore } = await import('../stores/sessionStore')
      const store = useSessionStore()
      store.setSessionID('test-session-id')

      websocketClient.connect('session-1')
      const firstWs = MockWebSocket.instances[0]!
      firstWs.simulateOpen()

      // connect() replaces the socket; the old socket's close event arrives
      // asynchronously AFTER the new connection already exists
      websocketClient.connect('session-2')
      expect(firstWs.readyState).toBe(3)

      // The stale close event must not schedule a reconnect
      vi.advanceTimersByTime(60000)
      expect(MockWebSocket.instances).toHaveLength(2)
    })

    it('should reconnect with exponential backoff', async () => {
      const { useSessionStore } = await import('../stores/sessionStore')
      const store = useSessionStore()
      store.setSessionID('test-session-id')

      websocketClient.connect('test-session-id')

      // 1st unexpected close: reconnect after 3s (2^0 * 3000)
      const ws1 = MockWebSocket.instances[0]!
      ws1.readyState = 3
      ws1.onclose?.(new CloseEvent('close', { code: 1006 }))
      vi.advanceTimersByTime(2999)
      expect(MockWebSocket.instances).toHaveLength(1)
      vi.advanceTimersByTime(1)
      expect(MockWebSocket.instances).toHaveLength(2)

      // 2nd unexpected close: reconnect after 6s (2^1 * 3000)
      const ws2 = MockWebSocket.instances[1]!
      ws2.readyState = 3
      ws2.onclose?.(new CloseEvent('close', { code: 1006 }))
      vi.advanceTimersByTime(5999)
      expect(MockWebSocket.instances).toHaveLength(2)
      vi.advanceTimersByTime(1)
      expect(MockWebSocket.instances).toHaveLength(3)

      // 3rd unexpected close: reconnect after 12s (2^2 * 3000)
      const ws3 = MockWebSocket.instances[2]!
      ws3.readyState = 3
      ws3.onclose?.(new CloseEvent('close', { code: 1006 }))
      vi.advanceTimersByTime(11999)
      expect(MockWebSocket.instances).toHaveLength(3)
      vi.advanceTimersByTime(1)
      expect(MockWebSocket.instances).toHaveLength(4)
    })

    it('should stop reconnecting after max attempts', async () => {
      const { useSessionStore } = await import('../stores/sessionStore')
      const store = useSessionStore()
      store.setSessionID('test-session-id')

      websocketClient.connect('test-session-id')
      // attempts: 5 reconnects after 5 unexpected closes, then no more
      // delays: 3s, 6s, 12s, 24s, 48s (2^0..2^4 * 3000)
      for (let i = 0; i < 5; i++) {
        const ws = MockWebSocket.instances[i]!
        ws.readyState = 3
        ws.onclose?.(new CloseEvent('close', { code: 1006 }))
        vi.advanceTimersByTime(3000 * Math.pow(2, i))
      }
      expect(MockWebSocket.instances).toHaveLength(6)

      // 6th close must not schedule anything
      const ws6 = MockWebSocket.instances[5]!
      ws6.readyState = 3
      ws6.onclose?.(new CloseEvent('close', { code: 1006 }))
      vi.advanceTimersByTime(60000)
      expect(MockWebSocket.instances).toHaveLength(6)
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
