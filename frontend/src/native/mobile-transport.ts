import { configureApiTransport } from '../modules/session/api'
import { clearMobileToken, readMobileToken } from './session-token'
import { isNativeApp } from './platform'

export function configureMobileTransport(onUnauthorized: () => void): void {
  if (!isNativeApp()) return
  configureApiTransport({
    kind: 'mobile',
    getToken: readMobileToken,
    onUnauthorized: async () => {
      await clearMobileToken()
      onUnauthorized()
    },
  })
}
