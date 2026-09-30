import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NotFoundView from './NotFoundView.vue'

describe('NotFoundView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders 404 heading and return home button', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/404', component: NotFoundView },
        { path: '/login', component: { template: '<div>Login Page</div>' } },
      ],
    })
    await router.push('/404')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')

    render(NotFoundView, {
      global: {
        plugins: [router],
      },
    })

    expect(screen.getByRole('heading', { level: 1, name: 'Página no encontrada' })).toBeTruthy()
    expect(screen.getByText('404')).toBeTruthy()

    const button = screen.getByRole('button', { name: 'Volver al inicio' })
    expect(button).toBeTruthy()
    await fireEvent.click(button)

    expect(pushSpy).toHaveBeenCalledWith('/login')
  })
})
