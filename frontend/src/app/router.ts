import type { Pinia } from 'pinia'
import { createRouter, createWebHistory, type RouterHistory } from 'vue-router'
import { pinia as applicationPinia } from './pinia'
import LoginView from '../modules/auth/LoginView.vue'
import ChangePasswordView from '../modules/auth/ChangePasswordView.vue'
import JoinView from '../modules/invitations/JoinView.vue'
import NotFoundView from '../modules/not-found/NotFoundView.vue'
import OnboardingView from '../modules/onboarding/OnboardingView.vue'
import ForgotPasswordView from '../modules/password-reset/ForgotPasswordView.vue'
import ResetPasswordView from '../modules/password-reset/ResetPasswordView.vue'
import SessionHomeView from '../modules/session/SessionHomeView.vue'
import TeamView from '../modules/team/TeamView.vue'
import ClientsView from '../modules/clients/ClientsView.vue'
import OrdersListView from '../modules/orders/OrdersListView.vue'
import OrderCreateView from '../modules/orders/OrderCreateView.vue'
import OrderDetailView from '../modules/orders/OrderDetailView.vue'
import TrackingView from '../modules/tracking/TrackingView.vue'
// Sprint 3 - Lucero
import LandingView from '../modules/landing/LandingView.vue'
import TermsView from '../modules/legal/TermsView.vue'
import PrivacyView from '../modules/legal/PrivacyView.vue'
// Sprint 4 - Michael
import ConsoleView from '../modules/console/ConsoleView.vue'
// Sprint 4 - José
import SignupView from '../modules/signup/SignupView.vue'
import VerifyEmailView from '../modules/signup/VerifyEmailView.vue'
import { useOnboardingStore } from '../modules/onboarding/onboarding.store'
import { useSessionStore } from '../modules/session/session.store'

interface AppRouterOptions {
  pinia: Pinia
  history: RouterHistory
}

export const routes = [
  // Sprint 3 - Lucero
  { path: '/', name: 'landing', component: LandingView, meta: { public: true } },
  { path: '/terminos', name: 'terms', component: TermsView, meta: { public: true, allowAuthenticated: true } },
  { path: '/privacidad', name: 'privacy', component: PrivacyView, meta: { public: true, allowAuthenticated: true } },
  { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
  { path: '/join', name: 'join', component: JoinView, meta: { public: true } },
  { path: '/onboarding', name: 'onboarding', component: OnboardingView, meta: { public: true } },
  { path: '/forgot-password', name: 'forgot-password', component: ForgotPasswordView, meta: { public: true } },
  { path: '/reset-password', name: 'reset-password', component: ResetPasswordView, meta: { public: true, allowAuthenticated: true } },
  { path: '/change-password', name: 'change-password', component: ChangePasswordView },
  { path: '/seguimiento/:token', name: 'tracking', component: TrackingView, meta: { public: true, allowAuthenticated: true } },
  { path: '/consola', name: 'console', component: ConsoleView, meta: { public: true, allowAuthenticated: true } },
  // Sprint 4 - José
  { path: '/registro', name: 'signup', component: SignupView, meta: { public: true } },
  { path: '/verificar-correo', name: 'verify-email', component: VerifyEmailView, meta: { public: true, allowAuthenticated: true } },
  { path: '/app', name: 'app', component: SessionHomeView },
  { path: '/app/production', redirect: '/app/orders' },
  { path: '/app/clients', name: 'clients', component: ClientsView, meta: { roles: ['OWNER', 'ADMIN'] } },
  { path: '/app/orders', name: 'orders', component: OrdersListView },
  { path: '/app/orders/new', name: 'orders-new', component: OrderCreateView, meta: { roles: ['OWNER', 'ADMIN'] } },
  { path: '/app/orders/:id', name: 'orders-detail', component: OrderDetailView },
  { path: '/app/team', name: 'team', component: TeamView, meta: { roles: ['OWNER', 'ADMIN'] } },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView, meta: { public: true } },
]

export function createAppRouter({ pinia, history }: AppRouterOptions) {
  const router = createRouter({ history, routes })

  router.beforeEach(async (to) => {
    const session = useSessionStore(pinia)
    await session.restore()

    if (to.name === 'not-found' || to.meta.allowAuthenticated) {
      return
    }

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
        // Once the first workshop exists, "register" means open signup.
        if (!available && to.name === 'onboarding') return { name: 'signup' }
      } catch {
        // Keep the requested public screen usable when the availability probe is offline.
      }
    }

    const allowedRoles = to.meta.roles as string[] | undefined
    if (allowedRoles && !allowedRoles.includes(session.principal?.role ?? '')) {
      return '/app'
    }

    if (to.name === 'app' && session.principal?.role === 'OPERATOR') {
      return '/app/orders'
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
