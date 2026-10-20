import { afterEach, describe, expect, it, vi } from 'vitest'

const native = vi.hoisted(() => ({ value: true }))
const scanner = vi.hoisted(() => ({
  isGoogleBarcodeScannerModuleAvailable: vi.fn(), installGoogleBarcodeScannerModule: vi.fn(), scan: vi.fn(),
}))
vi.mock('./platform', () => ({ isNativeApp: () => native.value }))
vi.mock('@capacitor-mlkit/barcode-scanning', () => ({
  BarcodeScanner: scanner,
  BarcodeFormat: { QrCode: 'QR_CODE', Code128: 'CODE_128', Code39: 'CODE_39' },
}))

import { canScanNatively, scanOrderCode } from './scanner'

afterEach(() => {
  native.value = true
  Object.values(scanner).forEach((mock) => mock.mockReset())
})

describe('scanOrderCode', () => {
  it('returns a scanned order code on Android', async () => {
    scanner.isGoogleBarcodeScannerModuleAvailable.mockResolvedValue({ available: true })
    scanner.scan.mockResolvedValue({ barcodes: [{ rawValue: ' OT-000003 ' }] })
    await expect(scanOrderCode()).resolves.toBe('OT-000003')
  })

  it('falls back to manual entry while the scanner module installs', async () => {
    scanner.isGoogleBarcodeScannerModuleAvailable.mockResolvedValue({ available: false })
    scanner.installGoogleBarcodeScannerModule.mockResolvedValue(undefined)
    await expect(scanOrderCode()).resolves.toBeNull()
    expect(scanner.scan).not.toHaveBeenCalled()
  })

  it('does not open a native scanner in the browser', async () => {
    native.value = false
    expect(canScanNatively()).toBe(false)
    await expect(scanOrderCode()).resolves.toBeNull()
    expect(scanner.scan).not.toHaveBeenCalled()
  })
})
