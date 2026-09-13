// Device tilt indicator for the camera mode.
// Primary source: DeviceOrientationEvent (beta/gamma) - how parallel the phone
// is held above the document. Display-up and flat on a table = 0 degrees deviation.
// Fallback source: Accelerometer (Generic Sensor API) when orientation events
// never deliver values (e.g. browsers that block motion sensors silently).

export type TiltStatus = 'good' | 'ok' | 'bad'
export type SensorUnavailableReason = 'unsupported' | 'no-events' | 'denied'

// Thresholds in degrees
export const TILT_GOOD = 6
export const TILT_OK = 15

const EMA_FACTOR = 0.15
const UNAVAILABLE_TIMEOUT_MS = 2000

interface DeviceOrientationEventConstructor {
  requestPermission?: () => Promise<'granted' | 'denied'>
}

// Minimal structural type for the Generic Sensor API so we do not depend
// on lib.dom shipping Accelerometer types.
interface MinimalSensor {
  start(): void
  stop(): void
  addEventListener(type: string, cb: (ev: Event) => void): void
  x: number | null
  y: number | null
  z: number | null
}
type AccelerometerCtor = new (opts?: { frequency?: number }) => MinimalSensor

function getAccelerometerCtor(): AccelerometerCtor | undefined {
  return (window as unknown as { Accelerometer?: AccelerometerCtor }).Accelerometer
}

export class AngleIndicator {
  private orientationHandler: ((e: DeviceOrientationEvent) => void) | null = null
  private sensor: MinimalSensor | null = null
  private fallbackTimers: number[] = []
  private smoothedDeg: number | null = null

  /**
   * Request sensor permission. MUST be called synchronously from a user
   * gesture (Safari drops the gesture context after the first await).
   * Returns true when permission is granted or not required.
   */
  async requestPermission(): Promise<boolean> {
    const ctor = window.DeviceOrientationEvent as unknown as DeviceOrientationEventConstructor | undefined
    if (ctor === undefined) return false
    if (typeof ctor.requestPermission === 'function') {
      try {
        const result = await ctor.requestPermission()
        return result === 'granted'
      } catch {
        return false
      }
    }
    // No permission flow needed (Android / desktop with sensor)
    return true
  }

  /**
   * Start the sensor chain: deviceorientation first, Accelerometer as
   * fallback. Each stage gets 2 seconds; the caller is notified when
   * every stage failed so it can switch to a non-sensor strategy.
   */
  start(onUpdate: (deg: number) => void, onUnavailable: (reason: SensorUnavailableReason) => void): void {
    this.stop()
    this.smoothedDeg = null

    if (typeof window.DeviceOrientationEvent === 'undefined') {
      this.tryAccelerometer(onUpdate, onUnavailable)
      return
    }

    this.orientationHandler = (e: DeviceOrientationEvent) => {
      // Ignore empty payloads (desktop browsers and blocked sensors send nulls)
      if (e.beta === null || e.gamma === null) return
      this.cancelFallback()

      const dev = Math.max(Math.abs(e.beta), Math.abs(e.gamma))
      if (this.smoothedDeg === null) {
        this.smoothedDeg = dev
      } else {
        this.smoothedDeg = this.smoothedDeg + EMA_FACTOR * (dev - this.smoothedDeg)
      }
      onUpdate(this.smoothedDeg)
    }

    window.addEventListener('deviceorientation', this.orientationHandler)
    this.scheduleFallback(() => this.tryAccelerometer(onUpdate, onUnavailable))
  }

  /** Stop listening and clean up all sources. */
  stop(): void {
    if (this.orientationHandler !== null) {
      window.removeEventListener('deviceorientation', this.orientationHandler)
      this.orientationHandler = null
    }
    if (this.sensor !== null) {
      this.sensor.stop()
      this.sensor = null
    }
    this.cancelFallback()
    this.smoothedDeg = null
  }

  private scheduleFallback(fn: () => void): void {
    this.fallbackTimers.push(window.setTimeout(fn, UNAVAILABLE_TIMEOUT_MS))
  }

  private cancelFallback(): void {
    this.fallbackTimers.forEach(t => window.clearTimeout(t))
    this.fallbackTimers = []
  }

  private tryAccelerometer(onUpdate: (deg: number) => void, onUnavailable: (reason: SensorUnavailableReason) => void): void {
    const Ctor = getAccelerometerCtor()
    if (Ctor === undefined) {
      onUnavailable('unsupported')
      return
    }

    let sensor: MinimalSensor
    try {
      sensor = new Ctor({ frequency: 10 })
    } catch {
      onUnavailable('unsupported')
      return
    }

    let gotReading = false
    sensor.addEventListener('error', (ev) => {
      const name = (ev as Event & { name?: string }).name
      this.stop()
      onUnavailable(name === 'NotAllowedError' ? 'denied' : 'no-events')
    })
    sensor.addEventListener('reading', () => {
      const s = this.sensor
      if (s === null || s.x === null || s.y === null || s.z === null) return
      if (!gotReading) {
        gotReading = true
        this.cancelFallback()
      }
      const deg = Math.atan2(Math.hypot(s.x, s.y), Math.abs(s.z)) * (180 / Math.PI)
      if (this.smoothedDeg === null) {
        this.smoothedDeg = deg
      } else {
        this.smoothedDeg = this.smoothedDeg + EMA_FACTOR * (deg - this.smoothedDeg)
      }
      onUpdate(this.smoothedDeg)
    })

    try {
      sensor.start()
    } catch {
      onUnavailable('unsupported')
      return
    }
    this.sensor = sensor
    this.scheduleFallback(() => {
      this.stopSensorOnly()
      onUnavailable('no-events')
    })
  }

  private stopSensorOnly(): void {
    if (this.sensor !== null) {
      this.sensor.stop()
      this.sensor = null
    }
  }
}

/** Map a deviation in degrees to a status bucket. */
export function tiltStatus(deg: number): TiltStatus {
  if (deg <= TILT_GOOD) return 'good'
  if (deg <= TILT_OK) return 'ok'
  return 'bad'
}
