import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import SignupView from './SignupView.vue'
import { signupApi } from './api'
import { ApiError } from '../session/api'

async function renderSignup() {
  const stub = { template: '<div />' }
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/registro', component: SignupView },
      { path: '/login', component: stub },
      { path: '/terminos', component: stub },
      { path: '/privacidad', component: stub },
    ],
  })
  await router.push('/registro')
  await router.isReady()
  render(SignupView, { global: { plugins: [createPinia(), router] } })
}

async function fillValidForm() {
  await fireEvent.update(screen.getByLabelText('Nombre del taller'), ' Confecciones Lucho ')
  await fireEvent.update(screen.getByLabelText('Tu nombre'), 'Lucho Quispe')
  await fireEvent.update(screen.getByLabelText('Correo electrónico'), ' lucho@taller.pe ')
  await fireEvent.update(screen.getByLabelText('Contraseña'), 'Secure signup password 2026!')
  await fireEvent.update(screen.getByLabelText('Repite tu contraseña'), 'Secure signup password 2026!')
}

afterEach(() => vi.restoreAllMocks())

describe('SignupView', () => {
  it('requires the terms and every field before calling the API', async () => {
    const register = vi.spyOn(signupApi, 'register')
    await renderSignup()

    await fireEvent.click(screen.getByRole('button', { name: 'Crear mi taller' }))

    expect(screen.getByText('Ingresa el nombre de tu taller.')).toBeTruthy()
    expect(screen.getByText('Ingresa un correo válido.')).toBeTruthy()
    expect(screen.getByText('Usa al menos 12 caracteres.')).toBeTruthy()
    expect(screen.getByText('Acepta los términos y la política de privacidad para continuar.')).toBeTruthy()
    expect(register).not.toHaveBeenCalled()
  })

  it('sends the contract fields and then asks the owner to check their email', async () => {
    const register = vi.spyOn(signupApi, 'register').mockResolvedValue({ data: {} })
    await renderSignup()
    await fillValidForm()
    await fireEvent.click(screen.getByRole('checkbox'))

    await fireEvent.click(screen.getByRole('button', { name: 'Crear mi taller' }))

    expect(register).toHaveBeenCalledWith({
      workshop_name: 'Confecciones Lucho', owner_name: 'Lucho Quispe', email: 'lucho@taller.pe',
      password: 'Secure signup password 2026!', accept_terms: true,
    })
    expect(await screen.findByRole('heading', { name: 'Revisa tu correo' })).toBeTruthy()
    expect(screen.getByText('lucho@taller.pe')).toBeTruthy()
  })

  it('resends the link and then waits before allowing another one', async () => {
    vi.spyOn(signupApi, 'register').mockResolvedValue({ data: {} })
    const resend = vi.spyOn(signupApi, 'resend').mockResolvedValue({ data: {} })
    await renderSignup()
    await fillValidForm()
    await fireEvent.click(screen.getByRole('checkbox'))
    await fireEvent.click(screen.getByRole('button', { name: 'Crear mi taller' }))

    await fireEvent.click(await screen.findByRole('button', { name: 'Reenviar el correo' }))

    expect(resend).toHaveBeenCalledWith('lucho@taller.pe')
    expect(await screen.findByText('Listo, te enviamos un enlace nuevo.')).toBeTruthy()
    const button = screen.getByRole('button', { name: /Reenviar en \d+ s/ }) as HTMLButtonElement
    expect(button.disabled).toBe(true)
  })

  it('explains when registration is closed', async () => {
    vi.spyOn(signupApi, 'register').mockRejectedValue(new ApiError(403, 'SIGNUP_CLOSED'))
    await renderSignup()
    await fillValidForm()
    await fireEvent.click(screen.getByRole('checkbox'))

    await fireEvent.click(screen.getByRole('button', { name: 'Crear mi taller' }))

    expect((await screen.findByRole('alert')).textContent).toContain('El registro de talleres está cerrado por ahora')
  })
})
