import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'pe.tallerflow.app',
  appName: 'TallerFlow',
  webDir: 'dist',
  android: { allowMixedContent: false },
}

export default config
