import type { Order } from './api'

export function orderNumberFromQuery(query: string): number | null {
  const match = query.trim().match(/^(?:OT[\s-]*)?0*(\d+)$/i)
  return match ? Number(match[1]) : null
}

export function matchesOrderSearch(order: Order, query: string): boolean {
  const text = query.trim().toLocaleLowerCase('es-PE')
  if (!text) return true
  const orderNumber = orderNumberFromQuery(text)
  if (orderNumber !== null) return order.order_number === orderNumber
  return [order.code, order.product, order.client_name, order.current_stage ?? '']
    .some((value) => value.toLocaleLowerCase('es-PE').includes(text))
}
