import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', route => {
    const path = new URL(route.request().url()).pathname
    const data = path.endsWith('/auth/session') ? { csrf_token: 'csrf', must_change_password: false }
      : path.endsWith('/me') ? { userId: 'u1', email: 'ana@example.test', displayName: 'Ana' }
        : path.endsWith('/workshops/current') ? { role: 'OWNER', workshop: { id: 'w1', name: 'Taller Demo' } }
          : path.endsWith('/clients') ? [
            { id: 'c1', name: 'Confecciones Gamarra y Asociados con nombre extenso', phone: '987654321', email: 'contacto.confecciones@example.test', is_active: true },
            { id: 'c2', name: 'Cliente sin contacto', phone: null, email: null, is_active: false },
          ] : []
    return route.fulfill({ json: { data } })
  })
})

for (const width of [360, 1366]) {
  test(`clients remain readable and searchable at ${width}px with one focus ring`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/app/clients')
    const client = page.getByRole('rowheader', { name: 'Confecciones Gamarra y Asociados con nombre extenso' })
    await expect(client).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    if (width === 360) {
      const nameWidth = await client.evaluate(el => el.getBoundingClientRect().width)
      expect(nameWidth).toBeGreaterThan(250)
    } else {
      const positions = await page.locator('.clients-list tbody tr').evaluateAll(rows => rows.map(row => Array.from(row.children).map(cell => Math.round(cell.getBoundingClientRect().x))))
      expect(positions[0]).toEqual(positions[1])
    }
    const search = page.getByRole('searchbox', { name: /buscar clientes/i })
    await search.focus()
    await expect(search).toBeFocused()
    expect(await page.locator('.search-bar').evaluate(el => getComputedStyle(el).outlineStyle)).toBe('solid')
    expect(await search.evaluate(el => getComputedStyle(el).outlineStyle)).toBe('none')
    expect(await search.evaluate(el => getComputedStyle(el).boxShadow)).toBe('none')
    await search.fill('sin contacto')
    await expect(page.getByRole('rowheader')).toHaveCount(1)
    await expect(page.getByText('Sin teléfono', { exact: true })).toBeVisible()
    await expect(page.getByText('Sin correo', { exact: true })).toBeVisible()
    await search.fill('no existe')
    await page.getByRole('button', { name: 'Ver todos los clientes' }).click()
    await expect(page.getByRole('rowheader')).toHaveCount(2)
  })
}
