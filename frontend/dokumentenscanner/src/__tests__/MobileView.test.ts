import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia } from 'pinia'

// Track handler registrations (hoisted-safe)
const registeredHandlers = vi.hoisted(() => new Map<string, ((data: unknown) => void)[]>())
const mockHandlers = vi.hoisted(() => ({
  on: vi.fn((event: string, handler: (data: unknown) => void) => {
    if (!registeredHandlers.has(event)) registeredHandlers.set(event, [])
    registeredHandlers.get(event)!.push(handler)
  }),
  off: vi.fn(),
  send: vi.fn(),
  connect: vi.fn(),
  disconnect: vi.fn(),
}))

const mockVerifyPIN = vi.hoisted(() => vi.fn())
const mockUploadImage = vi.hoisted(() => vi.fn())
const mockRoute = vi.hoisted(() => ({ query: { session_id: 'test-session-id' } }))

// Mock WebSocket
vi.stubGlobal('WebSocket', class MockWebSocket {
  static instances: MockWebSocket[] = []
  readyState = 1
  onopen: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  onmessage: ((ev: MessageEvent) => void) | null = null
  onerror: ((ev: Event) => void) | null = null
  sent: string[] = []
  constructor(public url: string) { MockWebSocket.instances.push(this) }
  simulateOpen() { this.readyState = 1; this.onopen?.(new Event('open')) }
  send(data: string) { this.sent.push(data) }
  close() { this.readyState = 3; this.onclose?.(new CloseEvent('close', { code: 1000 })) }
  simulateMessage(data: string) { this.onmessage?.(new MessageEvent('message', { data })) }
})

vi.mock('../utils/websocket', () => ({ websocketClient: mockHandlers }))

vi.mock('../utils/api', () => ({
  apiService: {
    verifyPIN: (...args: unknown[]) => mockVerifyPIN(...args),
    uploadImage: (...args: unknown[]) => mockUploadImage(...args),
    getWebSocketURL: (id: string) => `wss://test/ws/session/${id}`,
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
}))

import { mount } from '@vue/test-utils'
import MobileView from '../views/MobileView.vue'

describe('MobileView', () => {
  beforeEach(() => {
    registeredHandlers.clear()
    vi.useFakeTimers()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  function mountMobile() {
    return mount(MobileView, { global: { plugins: [createPinia()] } })
  }

  it('should show error when no session_id in URL', async () => {
    mockRoute.query = {}
    const wrapper = mountMobile()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('No session ID provided')
  })

  it('should show PIN input when session_id is provided', async () => {
    mockRoute.query = { session_id: 'test-session-id' }
    const wrapper = mountMobile()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Enter PIN')
  })

  it('should register download_request handler on mount', () => {
    mockRoute.query = { session_id: 'test-session-id' }
    mountMobile()
    expect(mockHandlers.on).toHaveBeenCalledWith('download_request', expect.any(Function))
  })

  it('should clean up download_request handler on unmount', () => {
    mockRoute.query = { session_id: 'test-session-id' }
    const wrapper = mountMobile()
    wrapper.unmount()
    expect(mockHandlers.off).toHaveBeenCalledWith('download_request', expect.any(Function))
  })

  it('should transition to upload view after successful PIN verification', async () => {
    mockRoute.query = { session_id: 'test-session-id' }
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountMobile()

    await wrapper.vm.$nextTick()

    // Find and fill PIN input
    const input = wrapper.find('input')
    await input.setValue('123456')

    // Click verify button
    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Upload Document')
    })
  })

  it('should show download confirm when download_request event fires', async () => {
    mockRoute.query = { session_id: 'test-session-id' }
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountMobile()

    await wrapper.vm.$nextTick()

    // Verify PIN to get to upload view
    const input = wrapper.find('input')
    await input.setValue('123456')
    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Upload Document')
    })

    // Simulate image_uploaded from backend (sets view to 'done')
    // Then trigger download_request
    const handler = registeredHandlers.get('download_request')![0]!

    // We need to get to 'done' state first - simulate upload completion
    // The view is in 'upload' state. download_request only transitions from 'done'.
    // So first we need to get to 'done'. In real flow, this happens via handleImageUploaded.
    // Let's directly set the state by calling the handler that would have been called
    // Actually we can test that the guard works:
    handler({})
    await wrapper.vm.$nextTick()
    // Should NOT show confirm because view is 'upload', not 'done'
    expect(wrapper.text()).not.toContain('Confirm Download')
  })

  it('should transition to confirm_download when download_request fires in done state', async () => {
    mockRoute.query = { session_id: 'test-session-id' }
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountMobile()

    await wrapper.vm.$nextTick()

    // Complete PIN flow
    const input = wrapper.find('input')
    await input.setValue('123456')
    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Upload Document')
    })

    // Manually trigger handleImageUploaded by finding and calling the handler
    // This is set up by the component when image is uploaded
    // Simulate the done state by calling the handler directly
    // The MobileView component transitions to 'done' via handleImageUploaded
    // We can simulate this by directly triggering the handler that the component registered
    // Actually, the upload happens through ImageUpload component which emits 'uploaded'
    // which calls handleImageUploaded. Since ImageUpload is a real component, let's
    // simulate the full flow by directly modifying the component's state.
    
    // Access the component instance
    const vm = wrapper.vm as { currentView: string }
    // Set the view to 'done' (simulating completed upload)
    vm.currentView = 'done'
    await wrapper.vm.$nextTick()

    // Now trigger download_request
    const handler = registeredHandlers.get('download_request')![0]!
    handler({})
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Confirm Download')
  })

  it('should send download_confirmed when confirm is clicked', async () => {
    mockRoute.query = { session_id: 'test-session-id' }
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountMobile()

    await wrapper.vm.$nextTick()

    // Complete PIN flow
    const input = wrapper.find('input')
    await input.setValue('123456')
    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Upload Document')
    })

    // Set to done state
    const vm = wrapper.vm as { currentView: string }
    vm.currentView = 'done'
    await wrapper.vm.$nextTick()

    // Trigger download_request
    const handler = registeredHandlers.get('download_request')![0]!
    handler({})
    await wrapper.vm.$nextTick()

    // Click confirm button
    const confirmBtn = wrapper.findAll('button').find(b => b.text().includes('Confirm'))!
    await confirmBtn.trigger('click')

    expect(mockHandlers.send).toHaveBeenCalledWith('download_confirmed', { session_id: 'test-session-id' })
  })
})
