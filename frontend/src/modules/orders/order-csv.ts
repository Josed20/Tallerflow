import type { Order } from './api'
import { canManageOperations } from '../session/role-policy'

/** Human-view CSV: normalize locale variants and prefix dangerous cells with a quoted tab.
 * https://community.owasp.org/attacks/CSV_Injection
 * These cells are intentionally text, not a lossless data interchange format.
 */
export function csvCell(value: unknown): string {
  const variants: Record<string, string> = { '＝': '=', '＋': '+', '－': '-', '＠': '@' }
  const text = String(value ?? '').replace(/[＝＋－＠]/gu, char => variants[char]!)
    .replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/gu, '')
  const dangerous = /^\s*[=+@-]/u.test(text) || /^[\t\r\n]/u.test(text)
  const safe = dangerous ? `\t${text.trimStart()}` : text
  return `"${safe.replace(/"/g, '""')}"`
}

export function ordersCsv(orders: readonly Order[], role: string | undefined): string {
  if (!canManageOperations(role)) throw new Error('Solo propietarios y administradores pueden exportar órdenes.')
  const status = { ACTIVE: 'Activa', COMPLETED: 'Completada', CANCELLED: 'Cancelada' }
  const traffic = { ON_TIME: 'A tiempo', AT_RISK: 'En riesgo', LATE: 'Atrasada', COMPLETED: 'Completada' }
  const units: Record<string, string> = { UNITS: 'Unidades', DOZENS: 'Docenas', KILOGRAMS: 'kg', METERS: 'm' }
  const rows = [
    ['Código', 'Cliente', 'Producto', 'Cantidad', 'Unidad', 'Estado', 'Semáforo', 'Estación', 'Entrega (ISO 8601)', 'Responsable (ID)', 'Responsable', 'Bloqueada', 'Motivo de bloqueo'],
    ...orders.map(order => [order.code, order.client_name, order.product, order.quantity, units[order.unit] ?? order.unit,
      status[order.status] ?? order.status, traffic[order.traffic_light] ?? order.traffic_light, order.current_stage ?? '', order.promised_at,
      order.responsible_user_id || 'Sin asignar', order.responsible_name || (order.responsible_user_id ? 'Sin nombre' : 'Sin asignar'), order.block_reason?.trim() ? 'Sí' : 'No', order.block_reason ?? '']),
  ]
  return '\uFEFF' + rows.map(row => row.map(csvCell).join(',')).join('\r\n') + '\r\n'
}

export function downloadOrdersCsv(orders: readonly Order[], role: string | undefined): void {
  const csv = ordersCsv(orders, role)
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `ordenes-${new Date().toISOString().slice(0, 10)}.csv`
  try {
    document.body.appendChild(link)
    link.click()
  } finally {
    link.remove()
    // Give the browser time to start reading the download before releasing its URL.
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }
}
