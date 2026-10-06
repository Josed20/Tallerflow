export type ClientPortalTrafficLight = 'ON_TIME' | 'AT_RISK' | 'LATE' | 'COMPLETED'
export type ClientPortalStageStatus = 'PENDING' | 'IN_PROGRESS' | 'COMPLETED'
export type ClientPortalOrderStatus = 'ACTIVE' | 'COMPLETED' | 'CANCELLED'

/** Data intentionally limited to what a customer may see. */
export interface ClientPortalStage {
  position: number
  name: string
  status: ClientPortalStageStatus
  completed_at: string | null
}

export interface ClientPortalOrder {
  id: string
  code: string
  product: string
  quantity: number
  unit: string
  status: ClientPortalOrderStatus
  traffic_light: ClientPortalTrafficLight
  promised_at: string
  updated_at: string
  stages?: ClientPortalStage[]
}
