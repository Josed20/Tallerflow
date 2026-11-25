import { expect, test, type Page, type Route } from '@playwright/test'
import type { Order } from '../src/modules/orders/api'

const workshopId = '11111111-1111-4111-8111-111111111111'
const userId = '22222222-2222-4222-8222-222222222222'
const orderId = '33333333-3333-4333-8333-333333333333'
const clientId = '44444444-4444-4444-8444-444444444444'
const stageNames = ['CORTE', 'COSTURA', 'ACABADO', 'CONTROL DE CALIDAD', 'PLANCHADO', 'EMPAQUE', 'ENTREGA']
const productionStages = stageNames.map(name => ({ name, client_description: `Tu pedido está en ${name.toLocaleLowerCase('es-PE')}.` }))
const firstClient = { id: clientId, workshop_id: workshopId, name: 'Confecciones Sur', is_active: true, email: null, phone: null, created_at: '2026-10-01T10:00:00Z', updated_at: '2026-10-01T10:00:00Z' }
const firstOrder: Order = {
  id: orderId, workshop_id: workshopId, client_id: clientId, client_name: firstClient.name,
  order_number: 1, code: 'OT-000001', product: 'Polos de algodón', specification: '', quantity: 48,
  unit: 'UNITS', status: 'ACTIVE', promised_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), version: 1,
  created_at: new Date(Date.now() - 48 * 60 * 60 * 1000).toISOString(), updated_at: new Date().toISOString(),
  traffic_light: 'AT_RISK', current_stage: 'CORTE', stage_count: stageNames.length, completed_stage_count: 0,
  responsible_user_id: null, responsible_name: '', block_reason: '',
}

async function mockWorkshop(page: Page, role: 'OWNER' | 'ADMIN' | 'OPERATOR' = 'OWNER') {
  let currentStageIndex = 0
  let savedOrder = { ...firstOrder }
  let completedAt: string | null = null
  const snapshot = (): Order => ({
    ...savedOrder,
    current_stage: stageNames[currentStageIndex], stage_count: stageNames.length, completed_stage_count: currentStageIndex,
    stages: productionStages.map((stage, index) => ({
      id: `55555555-5555-4555-8555-${String(index + 1).padStart(12, '0')}`,
      workshop_id: workshopId, order_id: orderId, position: index + 1, ...stage,
      status: index < currentStageIndex ? 'COMPLETED' : index === currentStageIndex ? 'IN_PROGRESS' : 'PENDING',
      completed_at: index < currentStageIndex ? completedAt : null,
      completed_by: index < currentStageIndex ? userId : null,
    })),
    photos: [], changes: [],
  })
  const assertMutation = (route: Route) => {
    expect(route.request().method()).toBe('PATCH')
    expect(route.request().headers()['x-csrf-token']).toBe('csrf-e2e')
  }
  await page.route('**/api/v1/auth/session', route => route.fulfill({ json: { data: { csrf_token: 'csrf-e2e', must_change_password: false } } }))
  await page.route('**/api/v1/me', route => route.fulfill({ json: { data: { userId, email: 'owner@example.test', displayName: 'Stephano' } } }))
  await page.route('**/api/v1/workshops/current', route => route.fulfill({ json: { data: { role, workshop: { id: workshopId, name: 'Taller Demo' } } } }))
  await page.route('**/api/v1/clients', route => route.fulfill({ json: { data: [firstClient] } }))
  await page.route('**/api/v1/team', route => route.fulfill({ json: { data: {
    members: [{ id: '66666666-6666-4666-8666-666666666666', user_id: userId, email: 'owner@example.test', display_name: 'Stephano', role, status: 'ACTIVE' }], invitations: [],
  } } }))
  await page.route('**/api/v1/production-settings', route => {
    expect(route.request().method()).toBe('GET')
    return route.fulfill({ json: { data: { version: 0, stages: productionStages } } })
  })
  await page.route('**/api/v1/dashboard', route => route.fulfill({ json: { data: { active: 1, on_time: 0, at_risk: 1, late: 0, completed: 0 } } }))
  await page.route('**/api/v1/orders', route => {
    const { stages: _stages, photos: _photos, changes: _changes, ...listedOrder } = snapshot()
    return route.fulfill({ json: { data: [listedOrder] } })
  })
  await page.route(`**/api/v1/orders/${orderId}`, route => route.fulfill({ json: { data: snapshot() } }))
  await page.route(`**/api/v1/orders/${orderId}/stages/1`, route => {
    assertMutation(route)
    expect(route.request().postDataJSON()).toEqual({ status: 'COMPLETED' })
    expect(currentStageIndex).toBe(0)
    currentStageIndex = 1
    completedAt = new Date().toISOString()
    savedOrder = { ...savedOrder, version: savedOrder.version + 1, updated_at: completedAt }
    return route.fulfill({ json: { data: snapshot().stages![0] } })
  })
  await page.route(`**/api/v1/orders/${orderId}/stages/1/reopen`, route => {
    expect(route.request().method()).toBe('POST')
    expect(route.request().headers()['x-csrf-token']).toBe('csrf-e2e')
    expect(route.request().postDataJSON().reason).toContain('error')
    expect(currentStageIndex).toBe(1)
    currentStageIndex = 0
    completedAt = null
    savedOrder = { ...savedOrder, version: savedOrder.version + 1, updated_at: new Date().toISOString() }
    return route.fulfill({ json: { data: snapshot().stages![0] } })
  })
  return { snapshot, update: (patch: Partial<Order>) => { savedOrder = { ...savedOrder, ...patch } } }
}

test('dashboard and Kanban work at mobile width', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  await page.goto('/app')
  await expect(page.getByRole('heading', { name: 'Taller Demo' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Equipo', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Trabajo por estación' })).toBeVisible()
  await expect(page.locator('.overview-tickets').getByText('Polos de algodón', { exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
  await page.screenshot({ path: testInfo.outputPath('dashboard-mobile.png'), fullPage: true })

  await page.getByRole('link', { name: /abrir tablero/i }).click()
  await expect(page.getByRole('heading', { name: 'Tablero de órdenes' })).toBeVisible()
  await expect(page.getByRole('region', { name: /corte, 1 órdenes/i })).toBeVisible()
  await page.getByRole('button', { name: 'Completar etapa', exact: true }).click()
  await expect(page.getByRole('dialog', { name: /terminó corte/i })).toBeVisible()
  await page.getByRole('button', { name: /seguir en esta etapa/i }).click()
  await expect(page.getByRole('region', { name: /corte, 1 órdenes/i })).toBeVisible()
  await page.getByRole('button', { name: 'Completar etapa', exact: true }).click()
  await expect(page.getByRole('dialog', { name: /terminó corte/i })).toContainText('La orden pasará a costura.')
  await page.getByRole('button', { name: /confirmar avance/i }).click()
  await expect(page.locator('.board-notice')).toContainText('pasó a costura')
  await expect(page.getByRole('region', { name: /costura, 1 órdenes/i })).toBeVisible()
  await expect.poll(() => page.locator('.kanban-scroll').evaluate((element) => element.scrollLeft)).toBeGreaterThan(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
  await page.screenshot({ path: testInfo.outputPath('board-mobile.png'), fullPage: true })
})

test('desktop list and filters keep actions reachable', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1366, height: 900 })
  await mockWorkshop(page)
  await page.goto('/app/orders?view=list')
  await expect(page.getByRole('heading', { name: 'Tablero de órdenes' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Lista', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: /en riesgo/i }).click()
  await expect(page.getByText('Polos de algodón', { exact: true })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('orders-desktop.png'), fullPage: true })
  await page.getByRole('button', { name: /atrasadas/i }).click()
  await expect(page.getByText(/no hay órdenes que coincidan/i)).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
})

test('delivery-today clears when switching risk filters', async ({ page }) => {
  await mockWorkshop(page)
  await page.goto('/app/orders?view=list&due=today')
  await expect(page.getByText('No hay órdenes que coincidan')).toBeVisible()
  await page.getByRole('button', { name: /todas 1/i }).click()
  await expect(page.getByText('Polos de algodón', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Entrega hoy' })).toHaveAttribute('aria-pressed', 'false')
  await page.getByRole('button', { name: /a tiempo 0/i }).click()
  await expect(page.getByText('No hay órdenes que coincidan')).toBeVisible()
  await page.getByRole('button', { name: /en riesgo 1/i }).click()
  await expect(page.getByText('Polos de algodón', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: /todas 1/i }).click()
  await page.getByRole('searchbox', { name: /buscar órdenes/i }).fill('1')
  await expect(page.getByText('OT-000001')).toBeVisible()
})

test('owner can correct the last stage from order detail', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  await page.goto('/app/orders')
  await page.getByRole('button', { name: 'Completar etapa', exact: true }).click()
  await expect(page.getByRole('dialog', { name: /terminó corte/i })).toContainText('La orden pasará a costura.')
  await page.getByRole('button', { name: /confirmar avance/i }).click()
  await expect(page.locator('.board-notice')).toContainText('pasó a costura')
  await page.getByRole('link', { name: /ver orden/i }).click()
  await expect(page.getByRole('button', { name: /corregir avance/i })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('order-detail-correction.png'), fullPage: true })
  await page.getByRole('button', { name: /corregir avance/i }).click()
  await page.getByLabel('Motivo de la corrección').fill('Se marcó por error')
  await page.getByRole('button', { name: 'Confirmar corrección' }).click()
  await expect(page.locator('.detail-notice[role="status"]')).toContainText('volvió a CORTE')
})

test('administrator can manage orders without owner-only navigation', async ({ page }) => {
  await mockWorkshop(page, 'ADMIN')
  await page.goto('/app/orders')
  await expect(page.locator('.board-header__actions').getByRole('link', { name: 'Nueva orden' })).toBeVisible()
  await page.getByRole('button', { name: 'Completar etapa', exact: true }).click()
  await expect(page.getByRole('dialog', { name: /terminó corte/i })).toContainText('La orden pasará a costura.')
  await page.getByRole('button', { name: /confirmar avance/i }).click()
  await page.getByRole('link', { name: /ver orden/i }).click()
  await expect(page.getByRole('button', { name: /corregir avance/i })).toBeVisible()
})

test('operator lands on orders and advances a stage from the order detail', async ({ page }, testInfo) => {
  await mockWorkshop(page, 'OPERATOR')
  await page.goto('/app')
  await expect(page).toHaveURL(/\/app\/orders$/)
  await expect(page.getByRole('heading', { name: 'Tablero de órdenes' })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Secciones internas' }).getByRole('link')).toHaveText(['Órdenes'])
  await expect(page.getByRole('link', { name: 'Nueva orden' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: /escanear ficha viajera/i })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('operator-orders-mobile.png'), fullPage: true })
  await page.getByRole('link', { name: 'Ver orden', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Polos de algodón' })).toBeVisible()
  await page.getByRole('button', { name: /completar etapa.*pasar a costura/i }).click()
  await expect(page.getByRole('dialog', { name: /terminó corte/i })).toContainText('La orden pasará a costura.')
  await page.getByRole('button', { name: /confirmar avance/i }).click()
  await expect(page.locator('.detail-notice[role="status"]')).toContainText('La orden pasó a COSTURA')
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
})

test('manager can reach the new-order form without a prefilled quantity', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  await page.goto('/app/orders/new')
  await expect(page.getByRole('heading', { name: 'Nueva orden de trabajo' })).toBeVisible()
  await expect(page.getByLabel(/cantidad/i)).toHaveValue('')
  await expect(page.getByRole('button', { name: 'Crear orden', exact: true })).toBeEnabled()
  await expect(page.getByText('La orden comenzará en CORTE.', { exact: false })).toBeVisible()
  await expect(page.getByLabel(/cliente/i)).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('new-order-mobile.png'), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
})

test('client directory stays usable on mobile and saves a contact correction', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  let phone: string | null = null
  await page.route('**/api/v1/clients', (route) => route.fulfill({ json: { data: [{ ...firstClient, phone, email: 'contacto@example.test' }] } }))
  await page.route(`**/api/v1/clients/${clientId}`, (route) => {
    expect(route.request().method()).toBe('PATCH')
    expect(route.request().headers()['x-csrf-token']).toBe('csrf-e2e')
    phone = route.request().postDataJSON().phone
    return route.fulfill({ json: { data: { ...firstClient, phone, email: 'contacto@example.test', updated_at: new Date().toISOString() } } })
  })
  await page.goto('/app/clients')
  await expect(page.getByRole('heading', { name: 'Clientes' })).toBeVisible()
  await expect(page.getByRole('rowheader', { name: 'Confecciones Sur', exact: true })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('clients-mobile.png'), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
  const directory = page.getByRole('table', { name: 'Clientes del taller' })
  const clientRow = directory.getByRole('row').filter({ has: page.getByRole('rowheader', { name: 'Confecciones Sur', exact: true }) })
  await expect(clientRow.getByText('Sin teléfono', { exact: true })).toBeVisible()
  await clientRow.getByRole('button', { name: 'Editar Confecciones Sur', exact: true }).click()
  await page.getByLabel('Teléfono / WhatsApp').fill('987654321')
  await page.getByRole('button', { name: 'Guardar cambios' }).click()
  await expect(page.locator('.clients-notice[role="status"]')).toContainText('se actualizó')
  await expect(clientRow.getByRole('link', { name: '987654321', exact: true })).toHaveAttribute('href', 'tel:987654321')
})

test('manager can edit an order from the order detail', async ({ page }, testInfo) => {
  const workshop = await mockWorkshop(page)
  const patchCapture: { data?: Record<string, unknown> } = {}
  await page.route(`**/api/v1/orders/${orderId}`, route => {
    if (route.request().method() === 'PATCH') {
      expect(route.request().headers()['x-csrf-token']).toBe('csrf-e2e')
      const body = route.request().postDataJSON() as Record<string, unknown>
      patchCapture.data = body
      expect(body.version).toBe(workshop.snapshot().version)
      workshop.update({
        client_id: String(body.client_id), product: String(body.product), quantity: Number(body.quantity),
        promised_at: String(body.promised_at), version: workshop.snapshot().version + 1, updated_at: new Date().toISOString(),
      })
    } else expect(route.request().method()).toBe('GET')
    return route.fulfill({ json: { data: workshop.snapshot() } })
  })

  await page.goto(`/app/orders/${orderId}`)
  await expect(page.getByRole('heading', { name: 'Polos de algodón' })).toBeVisible()
  await expect(page.getByRole('button', { name: /editar orden/i })).toBeVisible()
  await page.getByRole('button', { name: /editar orden/i }).click()

  await expect(page.getByRole('dialog', { name: /editar orden/i })).toBeVisible()
  await page.getByLabel(/prenda \/ producto/i).fill('Polos cuello camisero')
  await page.screenshot({ path: testInfo.outputPath('order-edit-dialog.png'), fullPage: true })
  await page.getByRole('button', { name: 'Guardar cambios' }).click()

  await expect(page.locator('.detail-notice[role="status"]')).toContainText('actualizados exitosamente')
  await expect(page.getByRole('heading', { name: 'Polos cuello camisero' })).toBeVisible()
  expect(patchCapture.data?.product).toBe('Polos cuello camisero')
  expect(patchCapture.data?.version).toBe(1)
})
