import { Camera, CameraResultType, CameraSource } from '@capacitor/camera'
import { isNativeApp } from './platform'

export async function compressPhoto(source: Blob): Promise<Blob> {
  const bitmap = await createImageBitmap(source)
  try {
    const scale = Math.min(1, 1600 / Math.max(bitmap.width, bitmap.height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(bitmap.width * scale))
    canvas.height = Math.max(1, Math.round(bitmap.height * scale))
    const context = canvas.getContext('2d')
    if (!context) throw new Error('No se pudo preparar la foto.')
    context.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
    return await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((blob) => blob ? resolve(blob) : reject(new Error('No se pudo comprimir la foto.')), 'image/jpeg', 0.8)
    })
  } finally {
    bitmap.close()
  }
}

function chooseWebPhoto(): Promise<File | null> {
  return new Promise((resolve) => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = 'image/*'
    input.style.display = 'none'
    document.body.append(input)
    let settled = false
    const done = (file: File | null) => {
      if (settled) return
      settled = true
      window.removeEventListener('focus', onFocus)
      input.remove()
      resolve(file)
    }
    const onFocus = () => window.setTimeout(() => {
      if (!input.files?.length) done(null)
    }, 500)
    input.addEventListener('change', () => done(input.files?.[0] ?? null), { once: true })
    window.addEventListener('focus', onFocus)
    input.click()
  })
}

export async function capturePhoto(): Promise<Blob | null> {
  if (!isNativeApp()) {
    const file = await chooseWebPhoto()
    return file ? compressPhoto(file) : null
  }
  try {
    const photo = await Camera.getPhoto({
      source: CameraSource.Camera,
      resultType: CameraResultType.Uri,
      quality: 85,
      saveToGallery: false,
    })
    if (!photo.webPath) return null
    return compressPhoto(await (await fetch(photo.webPath)).blob())
  } catch (error) {
    const message = String(error).toLowerCase()
    if (message.includes('cancel') || message.includes('denied') || message.includes('permission')) return null
    throw error
  }
}
