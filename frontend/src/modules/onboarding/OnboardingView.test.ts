import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import OnboardingView from './OnboardingView.vue'
import { useOnboardingStore } from './onboarding.store'

async function renderView() {
  const pinia = createPinia()
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/onboarding', component: OnboardingView }, { path: '/app', component: { template: '<div>App</div>' } },
  ] })
  await router.push('/onboarding')
  await router.isReady()
  render(OnboardingView, { global: { plugins: [pinia, router] } })
  return { pinia, router }
}

describe('OnboardingView', () => {
  it('shows all field errors and focuses the first invalid control', async () => {
    await renderView()

    await fireEvent.click(screen.getByRole('button', { name: 'Crear mi taller' }))

    expect(screen.getByText('Ingresa el nombre del taller.')).toBeTruthy()
    expect(screen.getByText('Ingresa tu nombre.')).toBeTruthy()
    expect(screen.getByText('Ingresa un correo válido.')).toBeTruthy()
    expect(screen.getByText('Usa al menos 12 caracteres.')).toBeTruthy()
    expect(document.activeElement).toBe(screen.getByLabelText('Nombre del taller'))
  })

  it('rejects a different password confirmation before submitting', async () => {
    const { pinia } = await renderView()
    const create = vi.spyOn(useOnboardingStore(pinia), 'create')
    await fillValidForm('Different password 456!')

    await fireEvent.click(screen.getByRole('button', { name: 'Crear mi taller' }))

    expect(screen.getByText('Las contraseñas no coinciden.')).toBeTruthy()
    expect(create).not.toHaveBeenCalled()
  })

  it('prevents duplicate submits and announces a generic server error', async () => {
    const { pinia } = await renderView()
    let rejectCreate!: (reason: unknown) => void
    const pending = new Promise<void>((_, reject) => { rejectCreate = reject })
    const create = vi.spyOn(useOnboardingStore(pinia), 'create').mockReturnValue(pending)
    await fillValidForm('Secure password 123!')
    const button = screen.getByRole('button', { name: 'Crear mi taller' })
    const form = button.closest('form') as HTMLFormElement

    await fireEvent.submit(form)
    await fireEvent.submit(form)

    expect(create).toHaveBeenCalledTimes(1)
    expect((button as HTMLButtonElement).disabled).toBe(true)
    rejectCreate(new Error('database secret must not render'))
    await pending.catch(() => undefined)
    const alert = await screen.findByRole('alert')
    expect(alert.getAttribute('aria-live')).toBe('assertive')
    expect(alert.textContent).toContain('No pudimos crear el taller')
    expect(alert.textContent).not.toContain('database secret')
  })
})

async function fillValidForm(confirmation: string) {
  await fireEvent.update(screen.getByLabelText('Nombre del taller'), 'Taller Web')
  await fireEvent.update(screen.getByLabelText('Tu nombre'), 'Owner Web')
  await fireEvent.update(screen.getByLabelText('Correo electrónico'), 'owner@web.test')
  await fireEvent.update(screen.getByLabelText('Contraseña'), 'Secure password 123!')
  await fireEvent.update(screen.getByLabelText('Confirmar contraseña'), confirmation)
}
