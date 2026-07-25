import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock WebSocket globally
class MockWebSocket {
  static instances: MockWebSocket[] = []
  static OPEN = 1
  static CLOSED = 3
  url: string
  readyState = 0
  onopen: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  onmessage: ((ev: MessageEvent) => void) | null = null
  onerror: ((ev: Event) => void) | null = null
  sent: string[] = []

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }
  simulateOpen() { this.readyState = 1; this.onopen?.(new Event('open')) }
  send(data: string) { this.sent.push(data) }
  close() { this.readyState = 3; this.onclose?.(new CloseEvent('close', { code: 1000 })) }
  simulateMessage(data: string) { this.onmessage?.(new MessageEvent('message', { data })) }
}
vi.stubGlobal('WebSocket', MockWebSocket)

// Track handler registrations
const registeredHandlers = new Map<string, ((data: unknown) => void)[]>()

// Mock the websocket module
vi.mock('../utils/websocket', () => ({
  websocketClient: {
    connect: vi.fn(),
    disconnect: vi.fn(),
    on: vi.fn((event: string, handler: (data: unknown) => void) => {
      if (!registeredHandlers.has(event)) registeredHandlers.set(event, [])
      registeredHandlers.get(event)!.push(handler)
    }),
    off: vi.fn((event: string, handler: (data: unknown) => void) => {
      const handlers = registeredHandlers.get(event)
      if (handlers) {
        const idx = handlers.indexOf(handler)
        if (idx !== -1) handlers.splice(idx, 1)
      }
    }),
    send: vi.fn(),
    isConnected: vi.fn(() => true),
    getReadyState: vi.fn(() => 1),
  },
}))

// Mock apiService
const mockCreateSession = vi.fn()
const mockDownloadPDF = vi.fn()
vi.mock('../utils/api', () => ({
  apiService: {
    createSession: (...args: unknown[]) => mockCreateSession(...args),
    downloadPDF: (...args: unknown[]) => mockDownloadPDF(...args),
    getWebSocketURL: (id: string) => `wss://test/ws/session/${id}`,
  },
}))

// Mock QRCodeDisplay
vi.mock('../components/QRCodeDisplay.vue', () => ({
  default: { name: 'QRCodeDisplay', template: '<div class="qr-mock">QR</div>' },
}))

import { mount } from '@vue/test-utils'
import { websocketClient } from '../utils/websocket'
import DesktopView from '../views/DesktopView.vue'

describe('DesktopView', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    registeredHandlers.clear()
    vi.useFakeTimers()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

const QRStub = { template: '<div class="qr-mock">QR</div>' }
const mountOpts = { global: { plugins: [createPinia()], stubs: { QRCodeDisplay: QRStub } } }

  it('should show loading state initially', () => {
    mockCreateSession.mockReturnValue(new Promise(() => {}))
    const wrapper = mount(DesktopView, mountOpts)
    expect(wrapper.text()).toContain('Initializing session')
  })

  it('should show QR display after session creation', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    const wrapper = mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(wrapper.find('.qr-mock').exists()).toBe(true)
    })
  })

  it('should show error state on session creation failure', async () => {
    mockCreateSession.mockRejectedValue(new Error('Network error'))
    const wrapper = mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Failed to create session')
    })
  })

  it('should call websocketClient.connect with session_id', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(websocketClient.connect).toHaveBeenCalledWith('test-id')
    })
  })

  it('should register image_added handler', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(websocketClient.on).toHaveBeenCalledWith('image_added', expect.any(Function))
    })
  })

  it('should register download_confirmed handler', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(websocketClient.on).toHaveBeenCalledWith('download_confirmed', expect.any(Function))
    })
  })

  it('should clean up handlers on unmount', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    const wrapper = mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(websocketClient.on).toHaveBeenCalled()
    })

    wrapper.unmount()

    expect(websocketClient.off).toHaveBeenCalledWith('image_added', expect.any(Function))
    expect(websocketClient.off).toHaveBeenCalledWith('download_confirmed', expect.any(Function))
    expect(websocketClient.disconnect).toHaveBeenCalled()
  })

  it('should call downloadPDF when download_confirmed handler fires', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    mockDownloadPDF.mockResolvedValue(new Blob(['pdf-data']))

    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:mock'),
      revokeObjectURL: vi.fn(),
    })

    const wrapper = mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(websocketClient.on).toHaveBeenCalledWith('download_confirmed', expect.any(Function))
    })

    const downloadHandler = registeredHandlers.get('download_confirmed')![0]!
    downloadHandler({})

    await vi.waitFor(() => {
      expect(mockDownloadPDF).toHaveBeenCalledWith('test-id')
    })
  })

  it('should send download_request when requestDownload is triggered', async () => {
    mockCreateSession.mockResolvedValue({ session_id: 'test-id', pin: '123456' })
    const wrapper = mount(DesktopView, mountOpts)

    await vi.waitFor(() => {
      expect(websocketClient.on).toHaveBeenCalled()
    })

    // Simulate page received
    const imageHandler = registeredHandlers.get('image_added')![0]!
    imageHandler({ page_count: 1 })
    await wrapper.vm.$nextTick()

    // Simulate PDF ready
    const pdfHandler = registeredHandlers.get('pdf_ready')![0]!
    pdfHandler({})
    await wrapper.vm.$nextTick()

    // Now the Download button should be visible
    const btn = wrapper.findAll('button').find(b => b.text().includes('Download'))!
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')

    expect(websocketClient.send).toHaveBeenCalledWith('download_request', { session_id: 'test-id' })
  })
})
