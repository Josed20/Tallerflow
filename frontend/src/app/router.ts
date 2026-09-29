import { createRouter, createWebHistory } from 'vue-router'
import { pinia } from './pinia'
import LoginView from '../modules/auth/LoginView.vue'
import ChangePasswordView from '../modules/auth/ChangePasswordView.vue'
import SessionHomeView from '../modules/session/SessionHomeView.vue'
import ForgotPasswordView from '../modules/password-reset/ForgotPasswordView.vue'
import ResetPasswordView from '../modules/password-reset/ResetPasswordView.vue'
import NotFoundView from '../modules/not-found/NotFoundView.vue'
import { useSessionStore } from '../modules/session/session.store'

export const routes = [
  { path: '/', redirect: '/app' },
  { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
  { path: '/forgot-password', name: 'forgot-password', component: ForgotPasswordView, meta: { public: true } },
  { path: '/reset-password', name: 'reset-password', component: ResetPasswordView, meta: { public: true, allowAuthenticated: true } },
  { path: '/change-password', name: 'change-password', component: ChangePasswordView },
  { path: '/app', name: 'app', component: SessionHomeView },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView, meta: { public: true } },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const session = useSessionStore(pinia)
  await session.restore()

  if (to.name === 'not-found') {
    return
  }

  if (to.meta.allowAuthenticated) {
    return
  }

  if (to.meta.public && session.isAuthenticated) {
    return session.requiresPasswordChange ? '/change-password' : '/app'
  }

  if (!to.meta.public && !session.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (session.requiresPasswordChange && to.name !== 'change-password') {
    return '/change-password'
  }

  if (!session.requiresPasswordChange && to.name === 'change-password') {
    return '/app'
  }
})
