import type { Pinia } from 'pinia'
import { createRouter, createWebHistory, type RouterHistory } from 'vue-router'
import { pinia as applicationPinia } from './pinia'
import LoginView from '../modules/auth/LoginView.vue'
import ChangePasswordView from '../modules/auth/ChangePasswordView.vue'
import OnboardingView from '../modules/onboarding/OnboardingView.vue'
import SessionHomeView from '../modules/session/SessionHomeView.vue'
import { useOnboardingStore } from '../modules/onboarding/onboarding.store'
import { useSessionStore } from '../modules/session/session.store'

interface AppRouterOptions {
  pinia: Pinia
  history: RouterHistory
}

export function createAppRouter({ pinia, history }: AppRouterOptions) {
  const router = createRouter({
    history,
    routes: [
      { path: '/', redirect: '/app' },
      { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
      { path: '/onboarding', name: 'onboarding', component: OnboardingView, meta: { public: true } },
      { path: '/change-password', name: 'change-password', component: ChangePasswordView },
      { path: '/app', name: 'app', component: SessionHomeView },
    ],
  })

  router.beforeEach(async (to) => {
    const session = useSessionStore(pinia)
    await session.restore()

    if (to.meta.public && session.isAuthenticated) {
      return session.requiresPasswordChange ? '/change-password' : '/app'
    }

    if (!to.meta.public && !session.isAuthenticated) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }

    if (!session.isAuthenticated && (to.name === 'login' || to.name === 'onboarding')) {
      try {
        const available = await useOnboardingStore(pinia).loadStatus()
        if (available && to.name === 'login') return { name: 'onboarding' }
        if (!available && to.name === 'onboarding') return { name: 'login' }
      } catch {
        // Keep the requested public screen usable when the availability probe is offline.
      }
    }

    if (session.requiresPasswordChange && to.name !== 'change-password') {
      return '/change-password'
    }

    if (!session.requiresPasswordChange && to.name === 'change-password') {
      return '/app'
    }
  })

  return router
}

export const router = createAppRouter({
  pinia: applicationPinia,
  history: createWebHistory(),
})
