import { render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'
import LoginView from './LoginView.vue'

describe('LoginView invited email', () => {
  it('prefills the email passed by the completed invitation flow', async () => {
    // Regression: ISSUE-001 — login discarded the email of the newly registered member.
    // Found by /qa on 2026-09-29
    // Report: .gstack/qa-reports/qa-report-localhost-2026-09-29.md
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/login', component: LoginView },
        { path: '/forgot-password', component: { template: '<main />' } },
      ],
    })
    await router.push('/login?email=new.member@example.test')
    await router.isReady()
    render(LoginView, { global: { plugins: [createPinia(), router] } })

    expect((screen.getByLabelText('Correo electrónico') as HTMLInputElement).value)
      .toBe('new.member@example.test')
  })
})
