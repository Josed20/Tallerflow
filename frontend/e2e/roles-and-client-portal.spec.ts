import { expect, test } from '@playwright/test'

test('client portal confirms a private-link request at a 360 px viewport', async ({ page }) => {
  await page.route('**/api/v1/client-portal/access-links', async (route) => {
    await route.fulfill({ status: 204 })
  })

  await page.goto('/portal/access')
  await page.getByLabel('Correo electrónico').fill('cliente@example.test')
  await page.getByRole('button', { name: 'Enviar enlace' }).click()

  await expect(page.getByText(/si el correo está asociado/i)).toBeVisible()
  await expect(page.getByText('cliente@example.test')).not.toBeVisible()
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
  expect(overflow).toBe(false)
})
