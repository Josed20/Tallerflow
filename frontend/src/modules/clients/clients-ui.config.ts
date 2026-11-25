import { defineConfig } from '@playwright/test'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

export default defineConfig({
  testDir: '.',
  testMatch: 'clients-ui.playwright.ts',
  outputDir: join(tmpdir(), 'tallerflow-clients-playwright-results'),
  workers: 1,
  reporter: 'list',
  use: { baseURL: process.env.E2E_BASE_URL ?? 'http://127.0.0.1:5179', browserName: 'chromium', channel: 'chrome', headless: true },
})
