import { createRouter, createWebHistory } from 'vue-router'
import { pinia } from './pinia'
import LoginView from '../modules/auth/LoginView.vue'
import ChangePasswordView from '../modules/auth/ChangePasswordView.vue'
import JoinView from '../modules/invitations/JoinView.vue'
import SessionHomeView from '../modules/session/SessionHomeView.vue'
import TeamView from '../modules/team/TeamView.vue'
import { useSessionStore } from '../modules/session/session.store'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/app' },
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    { path: '/join', name: 'join', component: JoinView, meta: { public: true } },
    { path: '/change-password', name: 'change-password', component: ChangePasswordView },
    { path: '/app', name: 'app', component: SessionHomeView },
    { path: '/app/team', name: 'team', component: TeamView, meta: { roles: ['OWNER', 'ADMIN'] } },
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

  const allowedRoles = to.meta.roles as string[] | undefined
  if (allowedRoles && !allowedRoles.includes(session.principal?.role ?? '')) {
    return '/app'
  }

  if (session.requiresPasswordChange && to.name !== 'change-password') {
    return '/change-password'
  }

  if (!session.requiresPasswordChange && to.name === 'change-password') {
    return '/app'
  }
})
