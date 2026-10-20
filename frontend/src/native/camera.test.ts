import { afterEach, describe, expect, it, vi } from 'vitest'

const native = vi.hoisted(() => ({ value: true }))
const getPhoto = vi.hoisted(() => vi.fn())
vi.mock('./platform', () => ({ isNativeApp: () => native.value }))
vi.mock('@capacitor/camera', () => ({
  Camera: { getPhoto }, CameraResultType: { Uri: 'uri' }, CameraSource: { Camera: 'camera' },
}))

import { capturePhoto } from './camera'

afterEach(() => {
  vi.unstubAllGlobals()
  getPhoto.mockReset()
  native.value = true
})

describe('capturePhoto', () => {
  it('compresses a native camera image to JPEG with a 1600px limit', async () => {
    getPhoto.mockResolvedValue({ webPath: 'capacitor://photo' })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ blob: async () => new Blob(['image'], { type: 'image/png' }) }))
    const close = vi.fn()
    vi.stubGlobal('createImageBitmap', vi.fn().mockResolvedValue({ width: 3200, height: 1600, close }))
    const drawImage = vi.fn()
    const originalCreate = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((name: string) => name === 'canvas'
      ? { width: 0, height: 0, getContext: () => ({ drawImage }), toBlob: (callback: (blob: Blob) => void) => callback(new Blob(['jpeg'], { type: 'image/jpeg' })) } as unknown as HTMLCanvasElement
      : originalCreate(name))

    const result = await capturePhoto()

    expect(result?.type).toBe('image/jpeg')
    expect(drawImage).toHaveBeenCalledWith(expect.anything(), 0, 0, 1600, 800)
    expect(close).toHaveBeenCalledOnce()
  })

  it('returns null when the camera is cancelled', async () => {
    getPhoto.mockRejectedValue(new Error('User cancelled'))
    await expect(capturePhoto()).resolves.toBeNull()
  })
})
