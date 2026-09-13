import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { AngleIndicator, tiltStatus, TILT_GOOD, TILT_OK } from './angleIndicator'

type OrientationHandler = (e: DeviceOrientationEvent) => void

// Fake DeviceOrientationEvent dispatch helper
let orientationHandler: OrientationHandler | null = null

function fireOrientation(beta: number | null, gamma: number | null) {
  const e = new Event('deviceorientation') as DeviceOrientationEvent
  Object.defineProperty(e, 'beta', { value: beta })
  Object.defineProperty(e, 'gamma', { value: gamma })
  orientationHandler?.(e)
}

// Fake Generic Sensor for the accelerometer fallback path
class FakeAccelerometer {
  static instances: FakeAccelerometer[] = []
  listeners: Record<string, (ev: Event) => void> = {}
  x: number | null = 0
  y: number | null = 0
  z: number | null = 9.81

  constructor(public opts?: { frequency?: number }) {}

  start() { FakeAccelerometer.instances.push(this) }
  stop() {}
  addEventListener(type: string, cb: (ev: Event) => void) { this.listeners[type] = cb }

  emitReading() { this.listeners['reading']?.(new Event('reading')) }
  emitError(name: string) {
    const e = new Event('error')
    Object.defineProperty(e, 'name', { value: name })
    this.listeners['error']?.(e)
  }
}

describe('tiltStatus thresholds', () => {
  it('returns good at or below TILT_GOOD', () => {
    expect(tiltStatus(0)).toBe('good')
    expect(tiltStatus(TILT_GOOD)).toBe('good')
  })

  it('returns ok between thresholds', () => {
    expect(tiltStatus(TILT_GOOD + 0.1)).toBe('ok')
    expect(tiltStatus(TILT_OK)).toBe('ok')
  })

  it('returns bad above TILT_OK', () => {
    expect(tiltStatus(TILT_OK + 0.1)).toBe('bad')
    expect(tiltStatus(90)).toBe('bad')
  })
})

describe('AngleIndicator', () => {
  const added: Record<string, OrientationHandler> = {}

  beforeEach(() => {
    vi.useFakeTimers()
    orientationHandler = null
    window.addEventListener = vi.fn((type: string, handler: EventListenerOrEventListenerObject) => {
      added[type] = handler as OrientationHandler
    })
    window.removeEventListener = vi.fn((type: string) => {
      delete added[type]
    })
    ;(window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent = {}
  })

  afterEach(() => {
    vi.useRealTimers()
    delete (window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent
    delete (window as unknown as { Accelerometer?: unknown }).Accelerometer
    FakeAccelerometer.instances = []
  })

  it('calls onUpdate with the first deviation unsmoothed', () => {
    const indicator = new AngleIndicator()
    const onUpdate = vi.fn()
    indicator.start(onUpdate, vi.fn())
    orientationHandler = added['deviceorientation'] ?? null
    fireOrientation(10, 20)
    expect(onUpdate).toHaveBeenCalledWith(20)
    indicator.stop()
  })

  it('smooths subsequent values with EMA', () => {
    const indicator = new AngleIndicator()
    const onUpdate = vi.fn()
    indicator.start(onUpdate, vi.fn())
    orientationHandler = added['deviceorientation'] ?? null
    fireOrientation(0, 20) // first: 20
    fireOrientation(0, 0) // 20 + 0.15 * (0 - 20) = 17
    expect(onUpdate).toHaveBeenLastCalledWith(17)
    indicator.stop()
  })

  it('uses the larger of beta/gamma as deviation', () => {
    const indicator = new AngleIndicator()
    const onUpdate = vi.fn()
    indicator.start(onUpdate, vi.fn())
    orientationHandler = added['deviceorientation'] ?? null
    fireOrientation(3, 8)
    expect(onUpdate).toHaveBeenLastCalledWith(8)
    indicator.stop()
  })

  it('calls onUnavailable when no events arrive within 2s', () => {
    const indicator = new AngleIndicator()
    const onUnavailable = vi.fn()
    indicator.start(vi.fn(), onUnavailable)
    orientationHandler = added['deviceorientation'] ?? null
    vi.advanceTimersByTime(2001)
    expect(onUnavailable).toHaveBeenCalledOnce()
    indicator.stop()
  })

  it('cancels the unavailable timer after the first event', () => {
    const indicator = new AngleIndicator()
    const onUnavailable = vi.fn()
    const onUpdate = vi.fn()
    indicator.start(onUpdate, onUnavailable)
    orientationHandler = added['deviceorientation'] ?? null
    fireOrientation(5, 5)
    vi.advanceTimersByTime(5000)
    expect(onUnavailable).not.toHaveBeenCalled()
    indicator.stop()
  })

  it('ignores events with null beta/gamma', () => {
    const indicator = new AngleIndicator()
    const onUpdate = vi.fn()
    const onUnavailable = vi.fn()
    indicator.start(onUpdate, onUnavailable)
    orientationHandler = added['deviceorientation'] ?? null
    fireOrientation(null, null)
    expect(onUpdate).not.toHaveBeenCalled()
    indicator.stop()
  })

  it('calls onUnavailable immediately when API is unsupported', () => {
    delete (window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent
    const indicator = new AngleIndicator()
    const onUnavailable = vi.fn()
    indicator.start(vi.fn(), onUnavailable)
    expect(onUnavailable).toHaveBeenCalledOnce()
  })

  it('removes the listener on stop', () => {
    const indicator = new AngleIndicator()
    indicator.start(vi.fn(), vi.fn())
    orientationHandler = added['deviceorientation'] ?? null
    indicator.stop()
    expect(added['deviceorientation']).toBeUndefined()
  })

  it('requestPermission returns false when API is missing', async () => {
    delete (window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent
    const indicator = new AngleIndicator()
    expect(await indicator.requestPermission()).toBe(false)
  })

  it('requestPermission returns true when no permission flow is required', async () => {
    const indicator = new AngleIndicator()
    expect(await indicator.requestPermission()).toBe(true)
  })

  it('requestPermission maps iOS granted result to true', async () => {
    ;(window as unknown as { DeviceOrientationEvent: { requestPermission: () => Promise<string> } }).DeviceOrientationEvent = {
      requestPermission: vi.fn().mockResolvedValue('granted'),
    }
    const indicator = new AngleIndicator()
    expect(await indicator.requestPermission()).toBe(true)
  })

  it('requestPermission maps iOS denied result to false', async () => {
    ;(window as unknown as { DeviceOrientationEvent: { requestPermission: () => Promise<string> } }).DeviceOrientationEvent = {
      requestPermission: vi.fn().mockResolvedValue('denied'),
    }
    const indicator = new AngleIndicator()
    expect(await indicator.requestPermission()).toBe(false)
  })
})

describe('AngleIndicator accelerometer fallback', () => {
  const added: Record<string, OrientationHandler> = {}

  beforeEach(() => {
    vi.useFakeTimers()
    orientationHandler = null
    FakeAccelerometer.instances = []
    window.addEventListener = vi.fn((type: string, handler: EventListenerOrEventListenerObject) => {
      added[type] = handler as OrientationHandler
    })
    window.removeEventListener = vi.fn((type: string) => {
      delete added[type]
    })
    ;(window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent = {}
    ;(window as unknown as { Accelerometer?: unknown }).Accelerometer = FakeAccelerometer
  })

  afterEach(() => {
    vi.useRealTimers()
    delete (window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent
    delete (window as unknown as { Accelerometer?: unknown }).Accelerometer
    FakeAccelerometer.instances = []
  })

  it('falls back to accelerometer when orientation only delivers nulls', () => {
    const indicator = new AngleIndicator()
    const onUpdate = vi.fn()
    const onUnavailable = vi.fn()
    indicator.start(onUpdate, onUnavailable)
    orientationHandler = added['deviceorientation'] ?? null
    fireOrientation(null, null)

    vi.advanceTimersByTime(2001)
    expect(FakeAccelerometer.instances).toHaveLength(1)

    FakeAccelerometer.instances[0]!.emitReading()
    expect(onUpdate).toHaveBeenCalledWith(0) // flat phone: atan2(0, 9.81) = 0
    expect(onUnavailable).not.toHaveBeenCalled()
    indicator.stop()
  })

  it('computes tilt from the accelerometer gravity vector', () => {
    const indicator = new AngleIndicator()
    const onUpdate = vi.fn()
    indicator.start(onUpdate, vi.fn())
    orientationHandler = added['deviceorientation'] ?? null
    vi.advanceTimersByTime(2001)

    const sensor = FakeAccelerometer.instances[0]!
    sensor.x = 9.81
    sensor.y = 0
    sensor.z = 0
    sensor.emitReading()
    expect(onUpdate).toHaveBeenCalledWith(90) // vertical phone

    indicator.stop()
  })

  it('maps accelerometer NotAllowedError to denied', () => {
    const indicator = new AngleIndicator()
    const onUnavailable = vi.fn()
    indicator.start(vi.fn(), onUnavailable)
    vi.advanceTimersByTime(2001)

    FakeAccelerometer.instances[0]!.emitError('NotAllowedError')
    expect(onUnavailable).toHaveBeenCalledWith('denied')
  })

  it('reports no-events when accelerometer never delivers readings', () => {
    const indicator = new AngleIndicator()
    const onUnavailable = vi.fn()
    indicator.start(vi.fn(), onUnavailable)
    vi.advanceTimersByTime(2001)
    expect(onUnavailable).not.toHaveBeenCalled()

    vi.advanceTimersByTime(2001)
    expect(onUnavailable).toHaveBeenCalledWith('no-events')
  })

  it('reports unsupported when neither source exists', () => {
    delete (window as unknown as { DeviceOrientationEvent?: unknown }).DeviceOrientationEvent
    delete (window as unknown as { Accelerometer?: unknown }).Accelerometer
    const indicator = new AngleIndicator()
    const onUnavailable = vi.fn()
    indicator.start(vi.fn(), onUnavailable)
    expect(onUnavailable).toHaveBeenCalledWith('unsupported')
  })
})
