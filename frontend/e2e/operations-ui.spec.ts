import { expect, test, type Page } from '@playwright/test'

const firstOrder = {
  id: 'order-1', workshop_id: 'workshop-1', client_id: 'client-1', client_name: 'Confecciones Sur',
  order_number: 1, code: 'OT-000001', product: 'Polos de algodón', specification: '', quantity: 48,
  unit: 'UNITS', status: 'ACTIVE', promised_at: '2026-10-15T18:00:00Z', version: 1,
  created_at: '2026-10-10T10:00:00Z', updated_at: '2026-10-10T10:00:00Z', traffic_light: 'AT_RISK', current_stage: 'CORTE',
}

async function mockWorkshop(page: Page, role: 'OWNER' | 'ADMIN' | 'OPERATOR' = 'OWNER') {
  let currentStage = 'CORTE'
  await page.route('**/api/v1/auth/session', (route) => route.fulfill({ json: { data: { csrf_token: 'csrf-e2e', must_change_password: false } } }))
  await page.route('**/api/v1/me', (route) => route.fulfill({ json: { data: { userId: 'owner-1', email: 'owner@example.test', displayName: 'Stephano' } } }))
  await page.route('**/api/v1/workshops/current', (route) => route.fulfill({ json: { data: { role, workshop: { id: 'workshop-1', name: 'Taller Demo' } } } }))
  await page.route('**/api/v1/dashboard', (route) => route.fulfill({ json: { data: { active: 1, on_time: 0, at_risk: 1, late: 0, completed: 0 } } }))
  await page.route('**/api/v1/orders', (route) => route.fulfill({ json: { data: [{ ...firstOrder, current_stage: currentStage }] } }))
  await page.route('**/api/v1/orders/order-1', (route) => route.fulfill({ json: { data: {
    ...firstOrder, current_stage: currentStage,
    stages: [
      { id: 'stage-1', workshop_id: 'workshop-1', order_id: 'order-1', position: 1, name: 'CORTE', status: currentStage === 'CORTE' ? 'IN_PROGRESS' : 'COMPLETED', completed_at: currentStage === 'CORTE' ? null : '2026-10-13T10:00:00Z' },
      { id: 'stage-2', workshop_id: 'workshop-1', order_id: 'order-1', position: 2, name: 'COSTURA', status: currentStage === 'COSTURA' ? 'IN_PROGRESS' : 'PENDING', completed_at: null },
    ],
  } } }))
  await page.route('**/api/v1/orders/order-1/stages/1', (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('csrf-e2e')
    currentStage = 'COSTURA'
    return route.fulfill({ json: { data: { id: 'stage-1', status: 'COMPLETED', position: 1 } } })
  })
  await page.route('**/api/v1/orders/order-1/stages/1/reopen', (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('csrf-e2e')
    expect(route.request().postDataJSON().reason).toContain('error')
    currentStage = 'CORTE'
    return route.fulfill({ json: { data: { id: 'stage-1', status: 'IN_PROGRESS', position: 1 } } })
  })
}

test('dashboard and Kanban work at mobile width', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  await page.goto('/app')
  await expect(page.getByRole('heading', { name: 'Taller Demo' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Equipo' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Trabajo por estación' })).toBeVisible()
  await expect(page.getByText('Polos de algodón')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
  await page.screenshot({ path: testInfo.outputPath('dashboard-mobile.png'), fullPage: true })

  await page.getByRole('link', { name: /abrir tablero/i }).click()
  await expect(page.getByRole('heading', { name: 'Tablero de órdenes' })).toBeVisible()
  await expect(page.getByRole('region', { name: /corte, 1 órdenes/i })).toBeVisible()
  await page.getByRole('button', { name: /pasar a costura/i }).click()
  await expect(page.getByRole('dialog', { name: /terminó corte/i })).toBeVisible()
  await page.getByRole('button', { name: /seguir en esta etapa/i }).click()
  await expect(page.getByRole('region', { name: /corte, 1 órdenes/i })).toBeVisible()
  await page.getByRole('button', { name: /pasar a costura/i }).click()
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
  await expect(page.getByText('Polos de algodón')).toBeVisible()
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
  await expect(page.getByText('Polos de algodón')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Entrega hoy' })).toHaveAttribute('aria-pressed', 'false')
  await page.getByRole('button', { name: /a tiempo 0/i }).click()
  await expect(page.getByText('No hay órdenes que coincidan')).toBeVisible()
  await page.getByRole('button', { name: /en riesgo 1/i }).click()
  await expect(page.getByText('Polos de algodón')).toBeVisible()
  await page.getByRole('button', { name: /todas 1/i }).click()
  await page.getByRole('searchbox', { name: /buscar órdenes/i }).fill('1')
  await expect(page.getByText('OT-000001')).toBeVisible()
})

test('owner can correct the last stage from order detail', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  await page.goto('/app/orders')
  await page.getByRole('button', { name: /pasar a costura/i }).click()
  await page.getByRole('button', { name: /confirmar avance/i }).click()
  await expect(page.locator('.board-notice')).toContainText('pasó a costura')
  await page.getByRole('link', { name: /ver orden/i }).click()
  await expect(page.getByRole('button', { name: /corregir avance/i })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('order-detail-correction.png'), fullPage: true })
  await page.getByRole('button', { name: /corregir avance/i }).click()
  await page.getByLabel('Motivo de la corrección').fill('Se marcó por error')
  await page.getByRole('button', { name: 'Confirmar corrección' }).click()
  await expect(page.getByRole('status')).toContainText('volvió a CORTE')
})

test('administrator can manage orders without owner-only navigation', async ({ page }) => {
  await mockWorkshop(page, 'ADMIN')
  await page.goto('/app/orders')
  await expect(page.locator('.board-header__actions').getByRole('link', { name: 'Nueva orden' })).toBeVisible()
  await page.getByRole('button', { name: /pasar a costura/i }).click()
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
  await page.getByRole('link', { name: /ver orden/i }).first().click()
  await expect(page.getByRole('heading', { name: 'Polos de algodón' })).toBeVisible()
  await page.getByRole('button', { name: /completar etapa.*pasar a costura/i }).click()
  await page.getByRole('button', { name: /confirmar avance/i }).click()
  await expect(page.getByRole('status')).toContainText('La orden pasó a COSTURA')
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
})

test('manager can reach the new-order form without a prefilled quantity', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  await page.route('**/api/v1/clients', (route) => route.fulfill({ json: { data: [{ id: 'client-1', workshop_id: 'workshop-1', name: 'Confecciones Sur', is_active: true, email: null, phone: null }] } }))
  await page.goto('/app/orders/new')
  await expect(page.getByRole('heading', { name: 'Nueva orden de trabajo' })).toBeVisible()
  await expect(page.getByLabel(/cantidad/i)).toHaveValue('')
  await expect(page.getByLabel(/cliente/i)).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('new-order-mobile.png'), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
})

test('client directory stays usable on mobile and saves a contact correction', async ({ page }, testInfo) => {
  await mockWorkshop(page)
  let phone: string | null = null
  await page.route('**/api/v1/clients', (route) => route.fulfill({ json: { data: [{ id: 'client-1', workshop_id: 'workshop-1', name: 'Confecciones Sur', phone, email: 'contacto@example.test', is_active: true }] } }))
  await page.route('**/api/v1/clients/client-1', (route) => {
    phone = route.request().postDataJSON().phone
    return route.fulfill({ json: { data: { id: 'client-1', workshop_id: 'workshop-1', name: 'Confecciones Sur', phone, email: 'contacto@example.test', is_active: true } } })
  })
  await page.goto('/app/clients')
  await expect(page.getByRole('heading', { name: 'Clientes' })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('clients-mobile.png'), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
  await page.getByRole('button', { name: 'Editar Confecciones Sur' }).click()
  await page.getByLabel('Teléfono / WhatsApp').fill('987654321')
  await page.getByRole('button', { name: 'Guardar cambios' }).click()
  await expect(page.getByRole('status')).toContainText('se actualizó')
  await expect(page.getByRole('link', { name: '987654321' })).toBeVisible()
})
