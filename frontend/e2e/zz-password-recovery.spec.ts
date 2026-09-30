import { expect, test } from '@playwright/test'

const ownerEmail = process.env.E2E_OWNER_EMAIL ?? 'owner.e2e@tallerflow.test'
const recoveredPassword = 'Recovered secure passphrase 11!'
const mailpitURL = process.env.E2E_MAILPIT_URL ?? 'http://127.0.0.1:8025'

test('password recovery delivers a one-time link, revokes sessions and preserves real 404s', async ({ page, request }) => {
  const unknown = await request.get('/this-route-must-not-exist')
  expect(unknown.status()).toBe(404)
  await expect(unknown.text()).resolves.toContain('Esta página no existe')

  await page.goto('/forgot-password')
  await page.getByLabel('Correo').fill(ownerEmail)
  await page.getByRole('button', { name: 'Enviar enlace' }).dblclick()
  await expect(page.getByText(/Si el correo existe/)).toBeVisible()

  const resetURL = await pollResetURL(request)
  await page.goto(resetURL)
  await page.getByLabel('Contraseña').fill(recoveredPassword)
  await page.getByRole('button', { name: 'Guardar contraseña' }).click()
  await expect(page.getByText('Contraseña actualizada. Ya puedes iniciar sesión.')).toBeVisible()

  const replay = await request.post('/api/v1/auth/password-resets/consume', {
    data: { Token: new URL(resetURL).searchParams.get('token'), Password: 'Replay secure passphrase 12!' },
  })
  expect(replay.status()).toBe(422)

  await page.goto('/login')
  await page.getByLabel('Correo').fill(ownerEmail)
  await page.getByLabel('Contraseña').fill(recoveredPassword)
  await page.getByRole('button', { name: 'Entrar' }).click()
  await expect(page).toHaveURL(/\/app$/)
})

async function pollResetURL(request: import('@playwright/test').APIRequestContext) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    const list = await request.get(`${mailpitURL}/api/v1/messages`)
    if (list.ok()) {
      const body = await list.json()
      const message = body.messages?.find((candidate: { To?: Array<{ Address?: string }> }) => candidate.To?.some(to => to.Address === ownerEmail))
      if (message?.ID) {
        const detail = await request.get(`${mailpitURL}/api/v1/message/${message.ID}`)
        const text = (await detail.json()).Text as string
        const match = text.match(/https?:\/\/[^\s]+\/reset-password\?token=[A-Za-z0-9_-]+/)
        if (match) return match[0]
      }
    }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  throw new Error('Mailpit did not receive the password reset link')
}
