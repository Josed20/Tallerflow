import { SecureStorage } from '@aparajita/capacitor-secure-storage'
import { isNativeApp } from './platform'

const TOKEN_KEY = 'tallerflow.mobile.session'

export async function readMobileToken(): Promise<string | null> {
  if (!isNativeApp()) return null
  const value = await SecureStorage.get(TOKEN_KEY)
  return typeof value === 'string' && value.length > 0 ? value : null
}

export async function writeMobileToken(token: string): Promise<void> {
  if (!isNativeApp()) return
  await SecureStorage.set(TOKEN_KEY, token)
}

export async function clearMobileToken(): Promise<void> {
  if (!isNativeApp()) return
  await SecureStorage.remove(TOKEN_KEY)
}
