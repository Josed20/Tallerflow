import { createApp } from 'vue'
import { VueQueryPlugin } from '@tanstack/vue-query'
import App from './App.vue'
import { pinia } from './app/pinia'
import { router } from './app/router'
import './styles/main.css'
import { configureMobileTransport } from './native/mobile-transport'
import { useSessionStore } from './modules/session/session.store'
import { App as NativeApp } from '@capacitor/app'
import { isNativeApp } from './native/platform'

configureMobileTransport(() => {
  useSessionStore(pinia).clearSession()
  void router.replace('/login?session=expired')
})

if (isNativeApp()) {
  void NativeApp.addListener('backButton', ({ canGoBack }) => {
    if (canGoBack) router.back()
    else void NativeApp.exitApp()
  })
}

createApp(App).use(pinia).use(router).use(VueQueryPlugin).mount('#app')
