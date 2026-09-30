import { expect, test } from '@playwright/test'

const ownerEmail = process.env.E2E_OWNER_EMAIL ?? 'owner.onboarding@tallerflow.test'
const ownerPassword = process.env.E2E_OWNER_PASSWORD ?? 'Permanent secure passphrase 27!'

test('an empty installation creates its permanent OWNER through the browser', async ({ page }) => {
  await page.goto('/login')
  await expect(page).toHaveURL(/\/onboarding$/)

  await page.getByRole('button', { name: 'Crear mi taller' }).click()
  await expect(page.getByText('Ingresa el nombre del taller.')).toBeVisible()
  await expect(page.getByText('Ingresa tu nombre.')).toBeVisible()
  await expect(page.getByText('Ingresa un correo válido.')).toBeVisible()
  await expect(page.getByText('Usa al menos 12 caracteres.')).toBeVisible()
  await expect(page.getByLabel('Nombre del taller')).toBeFocused()

  await page.getByLabel('Nombre del taller').fill('Taller Onboarding E2E')
  await page.getByLabel('Tu nombre').fill('Owner Onboarding')
  await page.getByLabel('Correo electrónico').fill(ownerEmail)
  await page.getByLabel('Contraseña', { exact: true }).fill(ownerPassword)
  await page.getByLabel('Confirmar contraseña').fill(ownerPassword)
  await page.getByRole('button', { name: 'Crear mi taller' }).click()

  await expect(page).toHaveURL(/\/app$/)
  await expect(page.getByRole('heading', { name: 'Taller Onboarding E2E' })).toBeVisible()
  await expect(page.getByText(`${ownerEmail} · OWNER`)).toBeVisible()

  const state = await page.evaluate(async ({ email, password }) => {
    const status = await fetch('/api/v1/onboarding/status')
    const duplicate = await fetch('/api/v1/onboarding/workshop', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workshop_name: 'Second workshop', owner_name: 'Second owner', email: `second-${email}`,
        password, password_confirmation: password,
      }),
    })
    const duplicateBody = await duplicate.text()
    return {
      status: await status.json(), duplicateStatus: duplicate.status, duplicateBody,
      local: Object.keys(localStorage), session: Object.keys(sessionStorage), cookie: document.cookie,
      viewportWidth: window.innerWidth, documentWidth: document.documentElement.scrollWidth,
    }
  }, { email: ownerEmail, password: ownerPassword })

  expect(state.status.data.available).toBe(false)
  expect(state.duplicateStatus).toBe(409)
  expect(state.duplicateBody).not.toContain(ownerPassword)
  expect(state.local).toEqual([])
  expect(state.session).toEqual([])
  expect(state.cookie).not.toContain('tallerflow_session')
  expect(state.documentWidth).toBeLessThanOrEqual(state.viewportWidth)

  await page.getByRole('button', { name: 'Cerrar sesión' }).click()
  await expect(page).toHaveURL(/\/login$/)
  await page.goto('/onboarding')
  await expect(page).toHaveURL(/\/login$/)
})
