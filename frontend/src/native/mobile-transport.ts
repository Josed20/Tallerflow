import { configureApiTransport } from '../modules/session/api'
import { clearMobileToken, readMobileToken } from './session-token'
import { usesNativeToken } from './platform'

export function configureMobileTransport(onUnauthorized: (reason: 'expired' | 'suspended') => void): void {
  if (!usesNativeToken()) return
  configureApiTransport({
    kind: 'mobile',
    getToken: readMobileToken,
    onUnauthorized: async (reason) => {
      await clearMobileToken()
      onUnauthorized(reason)
    },
  })
}
