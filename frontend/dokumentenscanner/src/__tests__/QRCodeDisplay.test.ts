import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('../utils/websocket', () => ({
  websocketClient: {
    connect: vi.fn(),
    disconnect: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
    send: vi.fn(),
  },
}))

import { mount } from '@vue/test-utils'
import { useSessionStore } from '../stores/sessionStore'
import QRCodeDisplay from '../components/QRCodeDisplay.vue'

describe('QRCodeDisplay', () => {
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  function mountWithSession(sessionId: string, pin: string) {
    const wrapper = mount(QRCodeDisplay, { global: { plugins: [pinia] } })
    const store = useSessionStore()
    store.setSessionID(sessionId)
    store.setPIN(pin)
    return wrapper
  }

  it('should show "Scan to Connect" heading', () => {
    const wrapper = mount(QRCodeDisplay, { global: { plugins: [pinia] } })
    expect(wrapper.text()).toContain('Scan to Connect')
  })

  it('should show waiting badge', () => {
    const wrapper = mount(QRCodeDisplay, { global: { plugins: [pinia] } })
    expect(wrapper.text()).toContain('Waiting for mobile upload')
  })

  it('should show PIN label', () => {
    const wrapper = mount(QRCodeDisplay, { global: { plugins: [pinia] } })
    expect(wrapper.text()).toContain('Or enter PIN manually')
  })

  it('should show spinner placeholder when no sessionID', () => {
    const wrapper = mount(QRCodeDisplay, { global: { plugins: [pinia] } })
    expect(wrapper.text()).toContain('Generating QR Code')
  })

  it('should display PIN value after session is set', async () => {
    const wrapper = mountWithSession('test-id', '123456')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('123456')
  })

  it('should render QR code image with correct src after session is set', async () => {
    const wrapper = mountWithSession('abc-123', '654321')
    await wrapper.vm.$nextTick()

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toContain('abc-123')
    expect(img.attributes('src')).toContain('/api/session/')
    expect(img.attributes('src')).toContain('/qrcode')
  })

  it('should display session ID', async () => {
    const wrapper = mountWithSession('test-session-id', '123456')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('test-session-id')
  })
})
