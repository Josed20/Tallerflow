import { expect, test } from '@playwright/test'

const token = 'A'.repeat(43)

test('customer follows an order from a private link at a 360 px viewport', async ({ page }) => {
  await page.route(`**/api/v1/tracking/${token}`, (route) => route.fulfill({
    json: {
      data: {
        workshop_name: 'Confecciones Gamarra', code: 'OT-000003', product: 'Polo jersey', quantity: 50, unit: 'DOZENS',
        status: 'ACTIVE', traffic_light: 'ON_TIME', promised_at: '2026-10-23T22:00:00Z', updated_at: '2026-10-13T15:00:00Z',
        stages: ['CORTE', 'COSTURA', 'ACABADO', 'CONTROL DE CALIDAD', 'PLANCHADO', 'EMPAQUE', 'ENTREGA'].map((name, index) => ({
          position: index + 1, name, status: index === 0 ? 'COMPLETED' : index === 1 ? 'IN_PROGRESS' : 'PENDING', completed_at: index === 0 ? '2026-10-12T15:00:00Z' : null,
        })),
        photos: [],
      },
    },
  }))

  await page.goto(`/seguimiento/${token}`)

  await expect(page.getByRole('heading', { name: 'Polo jersey' })).toBeVisible()
  await expect(page.getByText('En proceso ahora')).toBeVisible()
  await expect(page.getByText('1 de 7 etapas completadas')).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Secciones internas' })).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
})

test('customer sees a clear message for a replaced link', async ({ page }) => {
  await page.route(`**/api/v1/tracking/${token}`, (route) => route.fulfill({ status: 404, json: { error: { code: 'TRACKING_NOT_FOUND' } } }))
  await page.goto(`/seguimiento/${token}`)
  await expect(page.getByRole('heading', { name: 'Este enlace ya no está activo' })).toBeVisible()
})
