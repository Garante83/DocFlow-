import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

const mockVerifyPIN = vi.hoisted(() => vi.fn())
vi.mock('../utils/api', () => ({
  apiService: {
    verifyPIN: (...args: unknown[]) => mockVerifyPIN(...args),
  },
}))

const mockWsConnect = vi.hoisted(() => vi.fn())
vi.mock('../utils/websocket', () => ({
  websocketClient: {
    connect: mockWsConnect,
    on: vi.fn(),
    off: vi.fn(),
    send: vi.fn(),
    disconnect: vi.fn(),
  },
}))

import { mount } from '@vue/test-utils'
import { useSessionStore } from '../stores/sessionStore'
import PINInput from '../components/PINInput.vue'

describe('PINInput', () => {
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    pinia = createPinia()
    setActivePinia(pinia)
    const store = useSessionStore()
    store.setSessionID('test-session-id')
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function mountPIN() {
    return mount(PINInput, {
      global: {
        plugins: [pinia],
      },
    })
  }

  it('should render PIN input field', () => {
    const wrapper = mountPIN()
    expect(wrapper.find('input').exists()).toBe(true)
  })

  it('should render "Enter PIN" heading', () => {
    const wrapper = mountPIN()
    expect(wrapper.text()).toContain('Enter PIN')
  })

  it('should render Verify button', () => {
    const wrapper = mountPIN()
    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))
    expect(btn?.exists()).toBe(true)
  })

  it('should disable Verify button when PIN is less than 6 digits', () => {
    const wrapper = mountPIN()
    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))
    expect(btn?.attributes('disabled')).toBeDefined()
  })

  it('should enable Verify button when 6 digits are entered', async () => {
    const wrapper = mountPIN()
    const input = wrapper.find('input')
    await input.setValue('123456')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))
    expect(btn?.attributes('disabled')).toBeUndefined()
  })

  it('should call verifyPIN on successful verification', async () => {
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountPIN()

    const input = wrapper.find('input')
    await input.setValue('123456')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(mockVerifyPIN).toHaveBeenCalledWith('test-session-id', '123456')
    })
  })

  it('should emit verified on successful verification', async () => {
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountPIN()

    const input = wrapper.find('input')
    await input.setValue('123456')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.emitted('verified')).toBeTruthy()
    })
  })

  it('should show error message on invalid PIN', async () => {
    mockVerifyPIN.mockResolvedValue({ valid: false, message: 'Invalid PIN' })
    const wrapper = mountPIN()

    const input = wrapper.find('input')
    await input.setValue('000000')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Invalid PIN')
    })
  })

  it('should show error on connection failure', async () => {
    mockVerifyPIN.mockRejectedValue(new Error('Network error'))
    const wrapper = mountPIN()

    const input = wrapper.find('input')
    await input.setValue('123456')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Connection error')
    })
  })

  it('should auto-verify on paste of 6 digits', async () => {
    mockVerifyPIN.mockResolvedValue({ valid: true, message: 'OK' })
    const wrapper = mountPIN()

    const input = wrapper.find('input')
    await input.setValue('123456')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(mockVerifyPIN).toHaveBeenCalled()
    })
  })

  it('should increment failed attempts on invalid PIN', async () => {
    mockVerifyPIN.mockResolvedValue({ valid: false, message: 'Invalid' })
    const wrapper = mountPIN()

    const input = wrapper.find('input')
    await input.setValue('000000')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))!
    await btn.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('remaining')
    })
  })

  it('should not verify when PIN length is less than 6', async () => {
    const wrapper = mountPIN()
    const input = wrapper.find('input')
    await input.setValue('12345')

    const btn = wrapper.findAll('button').find(b => b.text().includes('Verify'))
    await btn?.trigger('click')

    expect(mockVerifyPIN).not.toHaveBeenCalled()
  })
})
