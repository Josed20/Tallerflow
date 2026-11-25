import { randomUUID } from 'node:crypto'
import { expect, test } from '@playwright/test'

test('workshop config preserves old orders and shares one recoverable customer link', async ({ page, context }, testInfo) => {
  test.skip(!process.env.E2E_OWNER_EMAIL, 'Requires an isolated installation with the onboarding owner.')
  const origin = new URL(testInfo.project.use.baseURL!).origin
  const login = await context.request.post('/api/v1/auth/login', {
    headers: { Origin: origin },
    data: { email: process.env.E2E_OWNER_EMAIL, password: process.env.E2E_OWNER_PASSWORD },
  })
  expect(login.ok()).toBeTruthy()
  const { data: { csrf_token: csrf } } = await login.json()
  const headers = { Origin: origin, 'X-CSRF-Token': csrf }
  const settingsResponse = await context.request.get('/api/v1/production-settings')
  expect(settingsResponse.ok()).toBeTruthy()
  const original = (await settingsResponse.json()).data
  const clientResponse = await context.request.post('/api/v1/clients', { headers, data: { name: `Cliente validación ${randomUUID().slice(0, 8)}` } })
  expect(clientResponse.status()).toBe(201)
  const client = (await clientResponse.json()).data
  async function createOrder(product: string) {
    const response = await context.request.post('/api/v1/orders', {
      headers: { ...headers, 'Idempotency-Key': randomUUID() },
      data: { client_id: client.id, product, quantity: 24, unit: 'UNITS', promised_at: new Date(Date.now() + 7 * 86400000).toISOString() },
    })
    expect(response.status()).toBe(201)
    return (await response.json()).data
  }
  const previous = await createOrder('Blusas con proceso original')
  const stages = Array.from({ length: 8 }, (_, i) => ({ name: `Etapa de prueba ${i + 1}`, client_description: `Descripción comprensible de la etapa ${i + 1}.` }))
  let changed = false
  try {
    const saved = await context.request.patch('/api/v1/production-settings', { headers, data: { version: original.version, stages } })
    expect(saved.ok()).toBeTruthy()
    changed = true
    const fresh = await createOrder('Blusas con proceso configurable')
    const freshDetail = (await (await context.request.get(`/api/v1/orders/${fresh.id}`)).json()).data
    expect(freshDetail.stages).toHaveLength(8)
    expect(freshDetail.stages[0].client_description).toBe(stages[0].client_description)
    const previousDetail = (await (await context.request.get(`/api/v1/orders/${previous.id}`)).json()).data
    expect(previousDetail.stages.map((stage: { name: string }) => stage.name)).toEqual(original.stages.map((stage: { name: string }) => stage.name))
    const blocked = await context.request.patch(`/api/v1/orders/${fresh.id}/operation`, { headers, data: { version: freshDetail.version, responsible_user_id: null, block_reason: 'Pendiente de aprobación de la muestra' } })
    expect(blocked.ok()).toBeTruthy()
    async function link() {
      const response = await context.request.post(`/api/v1/orders/${fresh.id}/tracking-link`, { headers })
      expect(response.ok()).toBeTruthy()
      return (await response.json()).data.token as string
    }
    const token = await link()
    expect(await link()).toBe(token)
    const publicResponse = await context.request.get(`/api/v1/tracking/${token}`)
    expect(publicResponse.ok()).toBeTruthy()
    const publicOrder = (await publicResponse.json()).data
    expect(publicOrder).not.toHaveProperty('block_reason')
    expect(publicOrder).not.toHaveProperty('responsible_user_id')
    await page.goto(`/seguimiento/${token}`)
    await expect(page.getByRole('heading', { name: fresh.product })).toBeVisible()
    await expect(page.getByText(stages[0].client_description).first()).toBeVisible()
    await page.getByRole('button', { name: /actualizar/i }).click()
    await expect(page.getByText('Estado actualizado.', { exact: true })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await page.screenshot({ path: testInfo.outputPath('seguimiento-configurable-mobile.png'), fullPage: true })
    await page.goto('/app/clients')
    await page.getByPlaceholder('Nombre, teléfono o correo').fill(client.name)
    await expect(page.getByText(client.name, { exact: true })).toBeVisible()
    await page.screenshot({ path: testInfo.outputPath('clientes-mobile.png'), fullPage: true })
    await page.goto('/app/account')
    await expect(page.getByRole('link', { name: 'Solicitar eliminación de cuenta' })).toBeVisible()
  } finally {
    if (changed) {
      const current = (await (await context.request.get('/api/v1/production-settings')).json()).data
      const restored = await context.request.patch('/api/v1/production-settings', { headers, data: { stages: original.stages, version: current.version } })
      expect(restored.ok()).toBeTruthy()
    }
  }
})
