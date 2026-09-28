import { expect, test } from '@playwright/test'

test.describe('Password Recovery & Route Handling', () => {
  test('navigates to real 404 page on unknown routes without redirecting to login', async ({ page }) => {
    await page.goto('/ruta-rara')
    await expect(page).toHaveURL(/\/ruta-rara$/)
    await expect(page.getByRole('heading', { level: 1, name: 'Página no encontrada' })).toBeVisible()
    await expect(page.getByText('404')).toBeVisible()

    await page.getByRole('button', { name: 'Volver al inicio' }).click()
    await expect(page).toHaveURL(/\/login$/)
  })

  test('redirects unauthenticated access from protected /app to /login with redirect query', async ({ page }) => {
    await page.goto('/app')
    await expect(page).toHaveURL(/\/login\?redirect=(%2F|\/)app$/)
  })

  test('navigates from login to forgot password via link', async ({ page }) => {
    await page.goto('/login')
    await page.getByRole('link', { name: '¿Olvidaste tu contraseña?' }).click()
    await expect(page).toHaveURL(/\/forgot-password$/)
    await expect(page.getByRole('heading', { level: 1, name: 'Recupera tu contraseña' })).toBeVisible()
  })

  test('submits password reset request with double click protection', async ({ page }) => {
    let requestsCount = 0
    await page.route('**/api/v1/auth/password-resets', async (route) => {
      requestsCount++
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            status: 'REQUESTED',
            message: 'Si el correo está registrado, recibirás un enlace para restablecer tu contraseña.',
          },
        }),
      })
    })

    await page.goto('/forgot-password')
    await page.getByLabel('Correo electrónico').fill('owner@tallerflow.pe')

    const submitBtn = page.getByRole('button', { name: 'Enviar enlace de recuperación' })
    // Rapid double click
    await submitBtn.click({ clickCount: 2, delay: 20 })

    await expect(page.getByRole('heading', { level: 1, name: 'Revisa tu correo' })).toBeVisible()
    expect(requestsCount).toBe(1)
  })

  test('shows invalid state when accessing reset password without token', async ({ page }) => {
    await page.goto('/reset-password')
    await expect(page.getByRole('heading', { level: 1, name: 'Enlace inválido' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Solicitar un nuevo enlace' })).toBeVisible()
  })

  test('completes password reset and redirects to login', async ({ page }) => {
    await page.route('**/api/v1/auth/password-resets/consume', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            status: 'PASSWORD_RESET_COMPLETED',
            message: 'Tu contraseña ha sido restablecida exitosamente.',
          },
        }),
      })
    })

    await page.goto('/reset-password?token=mocked-recovery-token-12345678901234567890')
    await expect(page.getByRole('heading', { level: 1, name: 'Crea tu nueva contraseña' })).toBeVisible()

    await page.getByLabel('Nueva contraseña', { exact: true }).fill('NewSecurePassword2026!')
    await page.getByLabel('Confirmar nueva contraseña').fill('NewSecurePassword2026!')

    await page.getByRole('button', { name: 'Restablecer contraseña' }).click()

    await expect(page.getByRole('heading', { level: 1, name: 'Contraseña restablecida' })).toBeVisible()
    await page.getByRole('button', { name: 'Iniciar sesión' }).click()
    await expect(page).toHaveURL(/\/login$/)
  })
})
