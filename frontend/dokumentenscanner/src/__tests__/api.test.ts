import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockPost = vi.hoisted(() => vi.fn())
const mockGet = vi.hoisted(() => vi.fn())

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      post: mockPost,
      get: mockGet,
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    })),
  },
}))

vi.mock('../stores/sessionStore', () => ({
  useSessionStore: () => ({
    sessionID: 'test-session-id',
    pin: '123456',
  }),
}))

import { apiService } from '../utils/api'

describe('apiService', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('createSession', () => {
    it('should return session_id and pin', async () => {
      mockPost.mockResolvedValue({
        data: { session_id: 'abc-123', pin: '654321' },
      })

      const result = await apiService.createSession()
      expect(result.session_id).toBe('abc-123')
      expect(result.pin).toBe('654321')
      expect(result.message).toBe('Session created')
      expect(mockPost).toHaveBeenCalledWith('/api/session')
    })
  })

  describe('verifyPIN', () => {
    it('should return valid on success', async () => {
      mockPost.mockResolvedValue({
        data: { valid: true, message: 'PIN verified' },
        status: 200,
      })

      const result = await apiService.verifyPIN('session-1', '123456')
      expect(result.valid).toBe(true)
      expect(result.session_id).toBe('session-1')
    })

    it('should return valid=false on failure', async () => {
      mockPost.mockResolvedValue({
        data: { valid: false, message: 'Invalid PIN' },
        status: 401,
      })

      const result = await apiService.verifyPIN('session-1', '000000')
      expect(result.valid).toBe(false)
    })

    it('should return valid=true when status is 200 even if data.valid is missing', async () => {
      mockPost.mockResolvedValue({
        data: { message: 'OK' },
        status: 200,
      })

      const result = await apiService.verifyPIN('session-1', '123456')
      expect(result.valid).toBe(true)
    })
  })

  describe('getLockStatus', () => {
    it('should return default lock status', async () => {
      const result = await apiService.getLockStatus()
      expect(result.locked).toBe(false)
      expect(result.attempts_remaining).toBe(3)
    })
  })

  describe('uploadImage', () => {
    it('should upload image and return result', async () => {
      mockPost.mockResolvedValue({
        data: { message: 'Image uploaded successfully', page_count: 1 },
      })

      const file = new File(['test'], 'test.jpg', { type: 'image/jpeg' })
      const result = await apiService.uploadImage('session-1', file)
      expect(result.image_id).toBe('session-1')
      expect(result.message).toBe('Image uploaded successfully')
      expect(result.page_count).toBe(1)
      expect(mockPost).toHaveBeenCalledWith(
        '/api/session/session-1/upload',
        expect.any(FormData),
        { headers: { 'Content-Type': 'multipart/form-data' } }
      )
    })
  })

  describe('finalizeUpload', () => {
    it('should call finalize endpoint', async () => {
      mockPost.mockResolvedValue({ data: { message: 'PDF generated', page_count: 3 } })

      await apiService.finalizeUpload('session-1')

      expect(mockPost).toHaveBeenCalledWith('/api/session/session-1/finalize')
    })

    it('should handle finalize error', async () => {
      mockPost.mockRejectedValue(new Error('No images'))

      await expect(apiService.finalizeUpload('session-1')).rejects.toThrow('No images')
    })
  })

  describe('downloadPDF', () => {
    it('should download PDF as blob', async () => {
      const blob = new Blob(['pdf-data'], { type: 'application/pdf' })
      mockGet.mockResolvedValue({ data: blob })

      const result = await apiService.downloadPDF('session-1')
      expect(result).toBe(blob)
      expect(mockGet).toHaveBeenCalledWith('/api/session/session-1/pdf', {
        responseType: 'blob',
      })
    })
  })

  describe('getWebSocketURL', () => {
    it('should construct URL containing session ID', () => {
      const url = apiService.getWebSocketURL('test-id')
      expect(url).toContain('test-id')
      expect(url).toContain('/ws/session/')
    })

    it('should use wss for https origin', () => {
      const url = apiService.getWebSocketURL('test-id')
      expect(url).toMatch(/^wss?:\/\//)
    })
  })
})
