// Visual tilt fallback for the camera mode.
// When device sensors are unavailable (blocked or missing), the live video
// stream is analyzed instead: the document region is separated from the
// background via Otsu thresholding and its keystone distortion estimates
// the tilt. Works for documents brighter OR darker than the background.
// All analysis happens client-side on a downscaled offscreen canvas.

import type { TiltStatus } from './angleIndicator'

// Keystone deviation thresholds (ratio units, 0 = perfectly rectangular)
export const VISUAL_GOOD = 0.1
export const VISUAL_OK = 0.25

// Filled-frame heuristic: document overflows the view or fills it uniformly
export const FILLED_BRIGHT_RATIO = 0.85
export const FILLED_MEAN_LUM = 130
const DEGENERATE_VARIANCE = 50

const MIN_DOC_AREA_RATIO = 0.08
const MIN_FILL = 0.4
const MIN_RUN_PX = 3
const EMA_FACTOR = 0.2
const SAMPLE_INTERVAL_MS = 250
const SAMPLE_WIDTH = 160

export type FrameAnalysisResult =
  | { kind: 'dev'; dev: number }
  | { kind: 'filled' }
  | { kind: 'nodoc' }

export type FrameStatus = TiltStatus | 'filled' | 'nodoc'

/** Map a keystone deviation to the shared status buckets. */
export function frameStatus(dev: number): TiltStatus {
  if (dev <= VISUAL_GOOD) return 'good'
  if (dev <= VISUAL_OK) return 'ok'
  return 'bad'
}

/**
 * Otsu threshold from a 256-bin histogram. Returns the separating threshold,
 * the between-class variance (0 = no separation possible) and the fraction of
 * pixels in the brighter class.
 */
function otsu(hist: Uint32Array, n: number): { threshold: number; variance: number; brightRatio: number } {
  let sumAll = 0
  for (let v = 0; v < 256; v++) sumAll += v * hist[v]!

  let wB = 0
  let sumB = 0
  let bestT = 0
  let bestVar = 0
  for (let t = 0; t < 256; t++) {
    wB += hist[t]!
    const wF = n - wB
    if (wB === 0 || wF === 0) continue
    sumB += t * hist[t]!
    const mB = sumB / wB
    const mF = (sumAll - sumB) / wF
    const between = (wB / n) * (wF / n) * (mB - mF) * (mB - mF)
    if (between > bestVar) {
      bestVar = between
      bestT = t
    }
  }

  let bright = 0
  for (let v = bestT + 1; v < 256; v++) bright += hist[v]!
  return { threshold: bestT, variance: bestVar, brightRatio: bright / n }
}

interface RegionMeasurement {
  dev: number
  fill: number
}

/**
 * Measure keystone distortion for one region class of the frame.
 * useBright selects the brighter class (lum > threshold), otherwise the
 * darker class. Returns null when the region is not rectangular enough.
 */
function measureRegion(
  lum: Uint8ClampedArray,
  w: number,
  h: number,
  useBright: boolean,
  threshold: number,
): RegionMeasurement | null {
  const rowRun = new Int32Array(h)
  const colRun = new Int32Array(w)
  let minX = w
  let minY = h
  let maxX = -1
  let maxY = -1

  for (let y = 0; y < h; y++) {
    const rowOff = y * w
    let run = 0
    for (let x = 0; x < w; x++) {
      const inRegion = useBright ? lum[rowOff + x]! > threshold : lum[rowOff + x]! <= threshold
      if (inRegion) {
        run++
        if (run > rowRun[y]!) rowRun[y] = run
        if (x - run + 1 < minX) minX = x - run + 1
        if (x > maxX) maxX = x
      } else {
        run = 0
      }
    }
  }

  for (let x = 0; x < w; x++) {
    let run = 0
    for (let y = 0; y < h; y++) {
      const inRegion = useBright ? lum[y * w + x]! > threshold : lum[y * w + x]! <= threshold
      if (inRegion) {
        run++
        if (run > colRun[x]!) colRun[x] = run
        if (y - run + 1 < minY) minY = y - run + 1
        if (y > maxY) maxY = y
      } else {
        run = 0
      }
    }
  }

  if (maxX < 0 || maxY < 0) return null

  const bboxW = maxX - minX + 1
  const bboxH = maxY - minY + 1

  // Rectangularity via mean longest runs vs. bbox size
  let rowSum = 0
  let validRows = 0
  for (let y = 0; y < h; y++) {
    if (rowRun[y]! >= MIN_RUN_PX) {
      rowSum += rowRun[y]!
      validRows++
    }
  }
  let colSum = 0
  let validCols = 0
  for (let x = 0; x < w; x++) {
    if (colRun[x]! >= MIN_RUN_PX) {
      colSum += colRun[x]!
      validCols++
    }
  }
  if (validRows === 0 || validCols === 0) return null
  const fill = ((rowSum / validRows) * (colSum / validCols)) / (bboxW * bboxH)
  if (fill < MIN_FILL) return null

  // Keystone: average region width in the top vs. bottom third of the bbox
  const yTop = minY + bboxH / 3
  const yBottom = minY + (2 * bboxH) / 3
  let topRows = 0
  let topSum = 0
  let bottomRows = 0
  let bottomSum = 0
  for (let y = minY; y <= maxY; y++) {
    const c = rowRun[y]!
    if (c < MIN_RUN_PX) continue
    if (y < yTop) {
      topRows++
      topSum += c
    } else if (y >= yBottom) {
      bottomRows++
      bottomSum += c
    }
  }
  if (topRows === 0 || bottomRows === 0) return null
  const vRatio = topSum / topRows / (bottomSum / bottomRows)

  // Roll: average region height in the left vs. right third of the bbox
  const xLeft = minX + bboxW / 3
  const xRight = minX + (2 * bboxW) / 3
  let leftCols = 0
  let leftSum = 0
  let rightCols = 0
  let rightSum = 0
  for (let x = minX; x <= maxX; x++) {
    const c = colRun[x]!
    if (c < MIN_RUN_PX) continue
    if (x < xLeft) {
      leftCols++
      leftSum += c
    } else if (x >= xRight) {
      rightCols++
      rightSum += c
    }
  }
  if (leftCols === 0 || rightCols === 0) return null
  const hRatio = leftSum / leftCols / (rightSum / rightCols)

  return { dev: Math.max(Math.abs(vRatio - 1), Math.abs(hRatio - 1)), fill }
}

/**
 * Analyze one RGBA frame. Emits a final status:
 * - 'filled': document fills/overflows the view (no edges measurable)
 * - 'nodoc': no usable document region (too dark, too small, too noisy)
 * - otherwise the keystone-based tilt bucket
 */
export function analyzeFrame(pixels: Uint8ClampedArray, w: number, h: number): FrameAnalysisResult {
  const n = w * h
  if (n === 0) return { kind: 'nodoc' }

  const lum = new Uint8ClampedArray(n)
  const hist = new Uint32Array(256)
  let lumSum = 0
  for (let i = 0; i < n; i++) {
    const r = pixels[i * 4]!
    const g = pixels[i * 4 + 1]!
    const b = pixels[i * 4 + 2]!
    const l = (r * 0.299 + g * 0.587 + b * 0.114) | 0
    lum[i] = l
    hist[l]!++
    lumSum += l
  }
  const meanLum = lumSum / n

  const { threshold: t, variance, brightRatio } = otsu(hist, n)
  const darkRatio = 1 - brightRatio

  // No meaningful split (uniform texture): judge by brightness only
  if (variance < DEGENERATE_VARIANCE) {
    return meanLum >= FILLED_MEAN_LUM ? { kind: 'filled' } : { kind: 'nodoc' }
  }

  const larger = Math.max(brightRatio, darkRatio)
  if (larger >= FILLED_BRIGHT_RATIO) {
    // A class overflows the frame; only trust it when it is the bright one
    return meanLum >= FILLED_MEAN_LUM ? { kind: 'filled' } : { kind: 'nodoc' }
  }
  if (Math.min(brightRatio, darkRatio) < MIN_DOC_AREA_RATIO) return { kind: 'nodoc' }

  // Try both orientations and keep the more rectangular region (document)
  const brightResult = measureRegion(lum, w, h, true, t)
  const darkResult = measureRegion(lum, w, h, false, t)
  const best =
    brightResult === null
      ? darkResult
      : darkResult === null
        ? brightResult
        : darkResult.fill > brightResult.fill
          ? darkResult
          : brightResult
  if (best === null) return { kind: 'nodoc' }
  return { kind: 'dev', dev: best.dev }
}

/** Runs the analysis loop on a live video element at a fixed sample rate. */
export class FrameAnalyzer {
  private canvas: HTMLCanvasElement | null = null
  private timer: number | null = null
  private smoothed: number | null = null

  start(video: HTMLVideoElement, onStatus: (status: FrameStatus) => void): void {
    this.stop()
    this.smoothed = null

    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    if (ctx === null) {
      onStatus('nodoc')
      return
    }
    this.canvas = canvas

    this.timer = window.setInterval(() => {
      if (video.readyState < 2 || video.videoWidth === 0) return

      const w = SAMPLE_WIDTH
      const h = Math.max(1, Math.round((video.videoHeight / video.videoWidth) * w))
      canvas.width = w
      canvas.height = h
      ctx.drawImage(video, 0, 0, w, h)
      const data = ctx.getImageData(0, 0, w, h).data

      const result = analyzeFrame(data, w, h)
      if (result.kind === 'nodoc') {
        this.smoothed = null
        onStatus('nodoc')
        return
      }
      if (result.kind === 'filled') {
        onStatus('filled')
        return
      }
      if (this.smoothed === null) {
        this.smoothed = result.dev
      } else {
        this.smoothed = this.smoothed + EMA_FACTOR * (result.dev - this.smoothed)
      }
      onStatus(frameStatus(this.smoothed))
    }, SAMPLE_INTERVAL_MS)
  }

  stop(): void {
    if (this.timer !== null) {
      window.clearInterval(this.timer)
      this.timer = null
    }
    this.canvas = null
    this.smoothed = null
  }
}
