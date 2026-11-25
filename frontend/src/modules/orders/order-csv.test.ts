import { afterEach, describe, expect, it, vi } from 'vitest'
import { csvCell, downloadOrdersCsv, ordersCsv } from './order-csv'
import type { Order } from './api'
const order = { code: 'OT-000001', client_name: '=1+1', product: 'Polos, "azul"\nL', quantity: 12, unit: 'UNITS', status: 'ACTIVE', traffic_light: 'ON_TIME', current_stage: 'CORTE', promised_at: '2026-11-24T18:00:00Z', phone: '999999999', tracking_token: 'secret' } as unknown as Order

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.useRealTimers() })
describe('order CSV safety and permissions', () => {
  it.each([
    ['=1+1', '=1+1'], ['+SUM(1)', '+SUM(1)'], ['-1+1', '-1+1'], ['@SUM(1)', '@SUM(1)'],
    ['  =1', '=1'], ['\t=1', '=1'], ['\r=1', '=1'], ['\n=1', '=1'], ['\u0000=1', '=1'],
    ['\uFEFF=1', '=1'], ['\ttext', 'text'], ['＝1+1', '=1+1'], ['＋SUM(1)', '+SUM(1)'],
    ['－1+1', '-1+1'], ['＠SUM(1)', '@SUM(1)'], [' \u0001\t＝1', '=1'], ['\u0085＋1', '+1'],
  ])('neutralizes %j as spreadsheet text', (value, normalized) => {
    expect(csvCell(value)).toBe(`"\t${normalized}"`)
  })
  it('removes non-printing controls and preserves safe text', () => {
    expect(csvCell('Po\u0000los')).toBe('"Polos"')
    expect(csvCell('  Taller Sur')).toBe('"  Taller Sur"')
  })
  it('quotes delimiters, quotes and line breaks', () => {
    expect(csvCell('Polos, "azul"\nL')).toBe('"Polos, ""azul""\nL"')
    expect(csvCell(null)).toBe('""')
  })
  it.each(['OWNER', 'ADMIN'])('exports only operational fields for %s', role => {
    const csv = ordersCsv([order], role)
    expect(csv.startsWith('\uFEFF"Código","Cliente"')).toBe(true)
    expect(csv).toContain(csvCell('=1+1'))
    expect(csv).toContain('"12","Unidades","Activa","A tiempo","CORTE"')
    expect(csv.endsWith('\r\n')).toBe(true)
    expect(csv).not.toMatch(/999999999|secret|WhatsApp|tracking_token/)
  })
  it('includes production responsibility and blocking with formula protection', () => {
    const csv = ordersCsv([{ ...order, responsible_user_id: 'user-production', responsible_name: '＝1+1', block_reason: '  =IMPORTDATA(1)' }], 'OWNER')
    expect(csv).toContain('"Responsable (ID)","Responsable","Bloqueada","Motivo de bloqueo"')
    expect(csv).toContain('"user-production",' + csvCell('＝1+1') + ',"Sí",' + csvCell('  =IMPORTDATA(1)'))
    expect(ordersCsv([order], 'OWNER')).toContain('"Sin asignar","Sin asignar","No",""')
  })
  it.each(['OPERATOR', '', undefined, 'owner'])('rejects role %j before download', role => {
    expect(() => ordersCsv([order], role)).toThrow(/propietarios y administradores/)
    expect(() => downloadOrdersCsv([order], role)).toThrow(/propietarios y administradores/)
  })
  it('downloads UTF-8 CSV and releases the URL', () => {
    vi.useFakeTimers()
    const createObjectURL = vi.fn((_blob: Blob) => 'blob:orders')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    downloadOrdersCsv([order], 'ADMIN')
    expect(click).toHaveBeenCalledOnce()
    expect(createObjectURL.mock.calls[0]?.[0]).toBeInstanceOf(Blob)
    expect(document.querySelector('a[download]')).toBeNull()
    vi.runAllTimers()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:orders')
  })
})
