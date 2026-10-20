import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'pe.tallerflow.app',
  appName: 'TallerFlow',
  webDir: 'dist',
  android: { allowMixedContent: false },
  ...(process.env.TF_ANDROID_LOCAL_PREVIEW === '1'
    ? { server: { url: 'http://localhost:8080', cleartext: true } }
    : {}),
}

export default config
