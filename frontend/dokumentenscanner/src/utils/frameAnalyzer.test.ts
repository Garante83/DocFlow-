import { describe, it, expect, vi } from 'vitest'
import { analyzeFrame, frameStatus, FrameAnalyzer, VISUAL_GOOD, VISUAL_OK } from './frameAnalyzer'

const W = 160
const H = 120

function makeFrame(luminanceAt: (x: number, y: number) => number): Uint8ClampedArray {
  const px = new Uint8ClampedArray(W * H * 4)
  for (let y = 0; y < H; y++) {
    for (let x = 0; x < W; x++) {
      const i = (y * W + x) * 4
      const l = luminanceAt(x, y)
      px[i] = l
      px[i + 1] = l
      px[i + 2] = l
      px[i + 3] = 255
    }
  }
  return px
}

// Centered bright rectangle on black background
const straightRect = makeFrame((x, y) => (x >= 20 && x < 140 && y >= 10 && y < 110 ? 255 : 0))

// Trapezoid: bright region widens from 100px (top) to 160px (bottom), centered at x=80
const trapezoid = makeFrame((x, y) => {
  if (y < 10 || y >= 110) return 0
  const width = 100 + (60 * (y - 10)) / 99
  return Math.abs(x - 79.5) <= width / 2 ? 255 : 0
})

// Grayish document with darker crease bands (folded mail) on black background
const creasedDoc = makeFrame((x, y) => {
  if (x >= 20 && x < 140 && y >= 10 && y < 110) {
    const crease = (y >= 45 && y < 55) || (y >= 80 && y < 86) ? 80 : 150
    return crease
  }
  return 0
})

// Dark document on bright background
const darkDoc = makeFrame((x, y) => (x >= 20 && x < 140 && y >= 10 && y < 110 ? 30 : 230))

// Document with uneven lighting gradient, filling most of the frame
const gradientDoc = makeFrame((x, y) => (x >= 10 && x < 150 && y >= 5 && y < 115 ? 100 + ((x - 10) * 80) / 139 : 0))

describe('frameStatus thresholds', () => {
  it('maps deviations to good/ok/bad', () => {
    expect(frameStatus(0)).toBe('good')
    expect(frameStatus(VISUAL_GOOD)).toBe('good')
    expect(frameStatus(VISUAL_GOOD + 0.01)).toBe('ok')
    expect(frameStatus(VISUAL_OK)).toBe('ok')
    expect(frameStatus(VISUAL_OK + 0.01)).toBe('bad')
  })
})

describe('analyzeFrame', () => {
  it('returns 0 deviation for a straight rectangular document', () => {
    expect(analyzeFrame(straightRect, W, H)).toEqual({ kind: 'dev', dev: 0 })
  })

  it('detects keystone distortion on a trapezoid document', () => {
    const result = analyzeFrame(trapezoid, W, H)
    expect(result.kind).toBe('dev')
    if (result.kind === 'dev') expect(result.dev).toBeGreaterThan(VISUAL_OK)
  })

  it('keeps creased documents as one region (Otsu separates paper from background)', () => {
    const result = analyzeFrame(creasedDoc, W, H)
    expect(result).toEqual({ kind: 'dev', dev: 0 })
  })

  it('recognizes a darker document on a bright background', () => {
    expect(analyzeFrame(darkDoc, W, H)).toEqual({ kind: 'dev', dev: 0 })
  })

  it('measures a textured gradient document with visible edges', () => {
    const result = analyzeFrame(gradientDoc, W, H)
    expect(result).toEqual({ kind: 'dev', dev: 0 })
  })

  it('returns filled for uniform bright frames', () => {
    expect(analyzeFrame(makeFrame(() => 210), W, H)).toEqual({ kind: 'filled' })
  })

  it('returns nodoc for uniform dark frames', () => {
    expect(analyzeFrame(makeFrame(() => 15), W, H)).toEqual({ kind: 'nodoc' })
  })

  it('returns nodoc when the document region is too small', () => {
    const tiny = makeFrame((x, y) => (x >= 70 && x < 100 && y >= 55 && y < 85 ? 255 : 0))
    expect(analyzeFrame(tiny, W, H)).toEqual({ kind: 'nodoc' })
  })

  it('returns nodoc for scattered bright noise (not rectangular)', () => {
    const noise = makeFrame((x, y) => ((x + y) % 3 === 0 ? 255 : 0))
    expect(analyzeFrame(noise, W, H)).toEqual({ kind: 'nodoc' })
  })

  it('returns nodoc for zero-size frames', () => {
    expect(analyzeFrame(new Uint8ClampedArray(0), 0, 0)).toEqual({ kind: 'nodoc' })
  })
})

describe('FrameAnalyzer', () => {
  it('emits nodoc when canvas 2d context is unavailable', () => {
    const realCreate = document.createElement.bind(document)
    const spy = vi.spyOn(document, 'createElement').mockImplementation(((tag: string) => {
      if (tag === 'canvas') return { getContext: () => null } as unknown as HTMLCanvasElement
      return realCreate(tag)
    }) as typeof document.createElement)

    const analyzer = new FrameAnalyzer()
    const onStatus = vi.fn()
    analyzer.start({} as HTMLVideoElement, onStatus)
    expect(onStatus).toHaveBeenCalledWith('nodoc')
    spy.mockRestore()
  })

  it('stop clears the loop without errors', () => {
    vi.useFakeTimers()
    const analyzer = new FrameAnalyzer()
    analyzer.start({} as HTMLVideoElement, vi.fn())
    analyzer.stop()
    vi.advanceTimersByTime(5000)
    vi.useRealTimers()
  })
})
