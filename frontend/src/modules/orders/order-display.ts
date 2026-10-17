export function formatOrderQuantity(quantity: number, unit: string): string {
  const amount = new Intl.NumberFormat('es-PE', { maximumFractionDigits: 2 }).format(quantity)
  const label = unit === 'UNITS' ? (quantity === 1 ? 'unidad' : 'unidades')
    : unit === 'DOZENS' ? (quantity === 1 ? 'docena' : 'docenas')
      : unit === 'KILOGRAMS' ? 'kg'
        : unit === 'METERS' ? 'm'
          : unit.toLocaleLowerCase('es-PE')
  return `${amount} ${label}`
}

export const productionStations = ['CORTE', 'COSTURA', 'ACABADO', 'CONTROL DE CALIDAD', 'PLANCHADO', 'EMPAQUE', 'ENTREGA']

/** Stations already finished, for the 7-stitch seam. */
export function completedStations(order: { status: string; current_stage?: string | null }): number {
  if (order.status === 'COMPLETED') return productionStations.length
  const index = productionStations.indexOf(order.current_stage?.toUpperCase() ?? '')
  return index < 0 ? 0 : index
}
