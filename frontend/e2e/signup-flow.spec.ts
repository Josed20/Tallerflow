import { expect, test, type APIRequestContext } from '@playwright/test'

// Needs the mail overlay (compose.mail.yaml): the verification link is read
// from Mailpit, exactly as an owner would open it from their inbox.
const mailpitURL = process.env.E2E_MAILPIT_URL ?? ''
const password = 'Permanent signup passphrase 27!'

async function verificationLink(request: APIRequestContext, email: string): Promise<string> {
  let link = ''
  await expect.poll(async () => {
    const search = await request.get(`${mailpitURL}/api/v1/search`, { params: { query: `to:${email}` } })
    const messages = (await search.json()).messages as { ID: string }[]
    if (!messages?.length) return ''
    const message = await request.get(`${mailpitURL}/api/v1/message/${messages[0].ID}`)
    link = (await message.json()).Text.match(/https?:\/\/\S+\/verificar-correo\?token=[\w-]{43}/)?.[0] ?? ''
    return link
  }, { timeout: 15_000, message: `no verification email for ${email}` }).not.toBe('')
  return link
}

test.describe('open signup', () => {
  test.skip(!mailpitURL, 'set E2E_MAILPIT_URL to the Mailpit API (needs compose.mail.yaml)')

  test('a new workshop registers, verifies its email and lands on its dashboard', async ({ page, request }) => {
    const email = `signup.${Date.now()}@tallerflow.test`
    const workshop = `Taller Registro ${Date.now()}`

    await page.goto('/registro')
    await page.getByRole('button', { name: 'Crear mi taller' }).click()
    await expect(page.getByText('Acepta los términos y la política de privacidad para continuar.')).toBeVisible()
    await expect(page.getByLabel('Nombre del taller')).toBeFocused()

    await page.getByLabel('Nombre del taller').fill(workshop)
    await page.getByLabel('Tu nombre').fill('Owner Registro')
    await page.getByLabel('Correo electrónico').fill(email)
    await page.getByLabel('Contraseña', { exact: true }).fill(password)
    await page.getByLabel('Repite tu contraseña').fill(password)
    await page.getByRole('checkbox').check()
    await page.getByRole('button', { name: 'Crear mi taller' }).click()
    await expect(page.getByRole('heading', { name: 'Revisa tu correo' })).toBeVisible()

    // Before verifying, the correct password is refused with a way forward.
    await page.goto('/login')
    await page.getByLabel('Correo electrónico').fill(email)
    await page.getByLabel('Contraseña').fill(password)
    await page.getByRole('button', { name: 'Iniciar sesión' }).click()
    await expect(page.getByText('Activa tu taller primero.')).toBeVisible()

    const link = await verificationLink(request, email)
    await page.goto(new URL(link).pathname + new URL(link).search)
    await expect(page).toHaveURL(/\/app$/)
    await expect(page.getByRole('heading', { name: workshop })).toBeVisible()

    // The same link cannot be used twice.
    await page.context().clearCookies()
    await page.goto(new URL(link).pathname + new URL(link).search)
    await expect(page.getByRole('heading', { name: 'Este enlace ya no sirve' })).toBeVisible()
    expect(await page.evaluate(() => window.location.search)).toBe('')

    // And the owner can now sign in normally.
    await page.goto('/login')
    await page.getByLabel('Correo electrónico').fill(email)
    await page.getByLabel('Contraseña').fill(password)
    await page.getByRole('button', { name: 'Iniciar sesión' }).click()
    await expect(page).toHaveURL(/\/app$/)

    const width = await page.evaluate(() => ({ viewport: window.innerWidth, document: document.documentElement.scrollWidth }))
    expect(width.document).toBeLessThanOrEqual(width.viewport)
  })
})
