export function formatOrderQuantity(quantity: number, unit: string): string {
  const amount = new Intl.NumberFormat('es-PE', { maximumFractionDigits: 2 }).format(quantity)
  const label = unit === 'UNITS' ? (quantity === 1 ? 'unidad' : 'unidades')
    : unit === 'DOZENS' ? (quantity === 1 ? 'docena' : 'docenas')
      : unit === 'KILOGRAMS' ? 'kg'
        : unit === 'METERS' ? 'm'
          : unit.toLocaleLowerCase('es-PE')
  return `${amount} ${label}`
}
