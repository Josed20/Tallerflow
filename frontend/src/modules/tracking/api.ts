export type TrackingTrafficLight = 'ON_TIME' | 'AT_RISK' | 'LATE' | 'COMPLETED'

/** Customer-facing order data returned by the public tracking endpoint. */
export interface TrackingOrder {
  workshop_name: string
  code: string
  product: string
  quantity: number
  unit: string
  status: 'ACTIVE' | 'COMPLETED' | 'CANCELLED'
  traffic_light: TrackingTrafficLight
  promised_at: string
  updated_at: string
  stages: { position: number; name: string; status: 'PENDING' | 'IN_PROGRESS' | 'COMPLETED'; completed_at: string | null }[]
  photos: { stage_position: number; photo_data: string; caption: string; created_at: string }[]
}

export class TrackingNotFoundError extends Error {}

export async function fetchTrackingOrder(token: string): Promise<TrackingOrder> {
  const response = await fetch(`/api/v1/tracking/${encodeURIComponent(token)}`, {
    headers: { Accept: 'application/json' },
    credentials: 'omit',
  })
  if (response.status === 404) throw new TrackingNotFoundError('Tracking link not found')
  if (!response.ok) throw new Error('Tracking request failed')
  const payload = await response.json() as { data: TrackingOrder }
  return payload.data
}
