import { Capacitor } from '@capacitor/core'

export function isNativeApp(): boolean {
  return Capacitor.isNativePlatform()
}

export function usesNativeToken(): boolean {
  return isNativeApp() && window.location.origin !== 'http://localhost:8080'
}
