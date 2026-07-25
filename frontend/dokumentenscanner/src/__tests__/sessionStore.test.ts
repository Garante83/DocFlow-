import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '../stores/sessionStore'

describe('sessionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('initial state', () => {
    it('should have null sessionID', () => {
      const store = useSessionStore()
      expect(store.sessionID).toBeNull()
    })

    it('should have waiting_for_pin status', () => {
      const store = useSessionStore()
      expect(store.status).toBe('waiting_for_pin')
    })

    it('should have empty pin', () => {
      const store = useSessionStore()
      expect(store.pin).toBe('')
    })

    it('should not be active', () => {
      const store = useSessionStore()
      expect(store.isActive).toBe(false)
    })

    it('should not be locked', () => {
      const store = useSessionStore()
      expect(store.isLocked).toBe(false)
    })
  })

  describe('setSessionID', () => {
    it('should set session ID and make store active', () => {
      const store = useSessionStore()
      store.setSessionID('abc-123')
      expect(store.sessionID).toBe('abc-123')
      expect(store.isActive).toBe(true)
    })
  })

  describe('setStatus', () => {
    it('should update status', () => {
      const store = useSessionStore()
      store.setStatus('upload_allowed')
      expect(store.status).toBe('upload_allowed')
    })
  })

  describe('setPIN', () => {
    it('should set PIN', () => {
      const store = useSessionStore()
      store.setPIN('123456')
      expect(store.pin).toBe('123456')
    })
  })

  describe('failed attempts', () => {
    it('should increment failed attempts', () => {
      const store = useSessionStore()
      store.incrementFailedAttempts()
      expect(store.failedAttempts).toBe(1)
      store.incrementFailedAttempts()
      expect(store.failedAttempts).toBe(2)
    })

    it('should reset failed attempts', () => {
      const store = useSessionStore()
      store.incrementFailedAttempts()
      store.incrementFailedAttempts()
      store.resetFailedAttempts()
      expect(store.failedAttempts).toBe(0)
    })
  })

  describe('lockout', () => {
    it('should not be locked when lockedUntil is null', () => {
      const store = useSessionStore()
      expect(store.isLocked).toBe(false)
    })

    it('should be locked when lockedUntil is in the future', () => {
      const store = useSessionStore()
      store.setLockedUntil(new Date(Date.now() + 60000))
      expect(store.isLocked).toBe(true)
    })

    it('should not be locked when lockedUntil is in the past', () => {
      const store = useSessionStore()
      store.setLockedUntil(new Date(Date.now() - 1000))
      expect(store.isLocked).toBe(false)
    })

    it('should accept string timestamps', () => {
      const store = useSessionStore()
      const future = new Date(Date.now() + 60000).toISOString()
      store.setLockedUntil(future)
      expect(store.isLocked).toBe(true)
    })

    it('should clear lock with null', () => {
      const store = useSessionStore()
      store.setLockedUntil(new Date(Date.now() + 60000))
      expect(store.isLocked).toBe(true)
      store.setLockedUntil(null)
      expect(store.isLocked).toBe(false)
    })
  })

  describe('reset', () => {
    it('should reset all state to defaults', () => {
      const store = useSessionStore()
      store.setSessionID('abc-123')
      store.setPIN('123456')
      store.setStatus('uploaded')
      store.incrementFailedAttempts()
      store.addImage(new File(['test'], 'test.jpg', { type: 'image/jpeg' }))
      store.setPDF('some-blob')

      store.reset()

      expect(store.sessionID).toBeNull()
      expect(store.status).toBe('waiting_for_pin')
      expect(store.pin).toBe('')
      expect(store.failedAttempts).toBe(0)
      expect(store.images).toHaveLength(0)
      expect(store.pdfData).toBeNull()
      expect(store.lockedUntil).toBeNull()
      expect(store.isActive).toBe(false)
    })
  })

  describe('addImage / removeImage', () => {
    it('should add images and track count', () => {
      const store = useSessionStore()
      expect(store.imageCount).toBe(0)
      expect(store.hasImages).toBe(false)

      store.addImage(new File(['a'], 'a.jpg', { type: 'image/jpeg' }))
      expect(store.imageCount).toBe(1)
      expect(store.hasImages).toBe(true)

      store.addImage(new File(['b'], 'b.jpg', { type: 'image/jpeg' }))
      expect(store.imageCount).toBe(2)
    })

    it('should remove image by index', () => {
      const store = useSessionStore()
      store.addImage(new File(['a'], 'a.jpg', { type: 'image/jpeg' }))
      store.addImage(new File(['b'], 'b.jpg', { type: 'image/jpeg' }))
      store.addImage(new File(['c'], 'c.jpg', { type: 'image/jpeg' }))

      store.removeImage(1)
      expect(store.imageCount).toBe(2)

      // First and third should remain
      expect(store.images[0].name).toBe('a.jpg')
      expect(store.images[1].name).toBe('c.jpg')
    })

    it('should handle out of bounds remove gracefully', () => {
      const store = useSessionStore()
      store.addImage(new File(['a'], 'a.jpg', { type: 'image/jpeg' }))
      store.removeImage(5)
      expect(store.imageCount).toBe(1)
    })

    it('should reorder images', () => {
      const store = useSessionStore()
      store.addImage(new File(['a'], 'a.jpg', { type: 'image/jpeg' }))
      store.addImage(new File(['b'], 'b.jpg', { type: 'image/jpeg' }))
      store.addImage(new File(['c'], 'c.jpg', { type: 'image/jpeg' }))

      store.reorderImage(0, 2)
      expect(store.images[0].name).toBe('b.jpg')
      expect(store.images[1].name).toBe('c.jpg')
      expect(store.images[2].name).toBe('a.jpg')
    })
  })
})
