import { expect, test } from '@playwright/test'

const ownerEmail = process.env.E2E_OWNER_EMAIL ?? 'owner.e2e@tallerflow.test'
const temporaryPassword = process.env.E2E_OWNER_PASSWORD ?? 'Temporary secure passphrase 9!'
const replacementPassword = process.env.E2E_NEW_PASSWORD ?? 'Replacement secure passphrase 10!'

test('owner completes the secure Sprint 1 authentication journey', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Correo electrónico').fill(ownerEmail)
  await page.getByLabel('Contraseña').fill('definitely-wrong-password')
  await page.getByLabel('Contraseña').press('Enter')
  await expect(page.getByRole('alert')).toContainText('No pudimos iniciar sesión')

  await page.getByLabel('Contraseña').fill(temporaryPassword)
  await page.getByLabel('Contraseña').press('Enter')
  await expect(page).toHaveURL(/\/change-password$/)
  await expect(page.getByText('Usa al menos 12 caracteres.')).toBeVisible()

  await page.getByLabel('Contraseña actual').fill(temporaryPassword)
  await page.getByLabel('Nueva contraseña').fill(replacementPassword)
  await page.getByLabel('Nueva contraseña').press('Enter')
  await expect(page).toHaveURL(/\/app$/)
  await expect(page.getByRole('heading', { name: 'Taller E2E' })).toBeVisible()
  await expect(page.getByText(`${ownerEmail} · OWNER`)).toBeVisible()

  const protectedResponses = await page.evaluate(async () => {
    const me = await fetch('/api/v1/me')
    const workshop = await fetch('/api/v1/workshops/current')
    return { me: { status: me.status, body: await me.json() }, workshop: { status: workshop.status, body: await workshop.json() } }
  })
  expect(protectedResponses.me.status).toBe(200)
  expect(protectedResponses.me.body.data.email).toBe(ownerEmail)
  expect(protectedResponses.workshop.status).toBe(200)
  expect(protectedResponses.workshop.body.data.workshop.name).toBe('Taller E2E')

  const csrfStatus = await page.evaluate(async () => (await fetch('/api/v1/auth/logout', { method: 'POST' })).status)
  expect(csrfStatus).toBe(403)
  await expect(page).toHaveURL(/\/app$/)

  const browserState = await page.evaluate(() => ({
    local: Object.keys(localStorage),
    session: Object.keys(sessionStorage),
    cookie: document.cookie,
  }))
  expect(browserState.local).toEqual([])
  expect(browserState.session).toEqual([])
  expect(browserState.cookie).not.toContain('tallerflow_session')

  await page.getByRole('button', { name: 'Cerrar sesión' }).click()
  await expect(page).toHaveURL(/\/login$/)
  const invalidSessionStatus = await page.evaluate(async () => (await fetch('/api/v1/auth/session')).status)
  expect(invalidSessionStatus).toBe(401)

  for (let attempt = 0; attempt < 4; attempt += 1) {
    const status = await loginRequest(page, `rotated-${attempt}@example.com`)
    expect(status.status).toBe(401)
  }
  const throttled = await loginRequest(page, 'rotated-final@example.com')
  expect(throttled.status).toBe(429)
  expect(throttled.retryAfter).toBe('900')
})

async function loginRequest(page: import('@playwright/test').Page, email: string) {
  return page.evaluate(async ({ email, password }) => {
    const response = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    return { status: response.status, retryAfter: response.headers.get('Retry-After') }
  }, { email, password: 'wrong-password' })
}
