import { BarcodeScanner, BarcodeFormat } from '@capacitor-mlkit/barcode-scanning'
import { isNativeApp } from './platform'

export function canScanNatively(): boolean {
  return isNativeApp()
}

export async function scanOrderCode(): Promise<string | null> {
  if (!canScanNatively()) return null
  try {
    const { available } = await BarcodeScanner.isGoogleBarcodeScannerModuleAvailable()
    if (!available) {
      await BarcodeScanner.installGoogleBarcodeScannerModule()
      return null
    }
    const result = await BarcodeScanner.scan({ formats: [BarcodeFormat.QrCode, BarcodeFormat.Code128, BarcodeFormat.Code39] })
    return result.barcodes[0]?.rawValue?.trim() || null
  } catch (error) {
    const message = String(error).toLowerCase()
    if (message.includes('cancel') || message.includes('permission') || message.includes('denied')) return null
    throw error
  }
}
