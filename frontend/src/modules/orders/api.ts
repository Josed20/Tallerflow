import { api } from '../session/api'

export type TrafficLight = 'ON_TIME' | 'AT_RISK' | 'LATE' | 'COMPLETED'
export type StageStatus = 'PENDING' | 'IN_PROGRESS' | 'COMPLETED'
export type OrderStatus = 'ACTIVE' | 'COMPLETED' | 'CANCELLED'

export interface OrderStage {
  id: string
  workshop_id: string
  order_id: string
  position: number
  name: string
  status: StageStatus
  completed_at: string | null
}

export interface StagePhoto {
  id: string
  workshop_id: string
  order_id: string
  stage_position: number
  photo_data: string
  caption: string
  created_at: string
}

export interface Order {
  id: string
  workshop_id: string
  client_id: string
  client_name: string
  order_number: number
  code: string
  product: string
  specification: string
  quantity: number
  unit: string
  status: OrderStatus
  promised_at: string
  version: number
  created_at: string
  updated_at: string
  traffic_light: TrafficLight
  current_stage?: string
  stages?: OrderStage[]
  photos?: StagePhoto[]
}

export interface DashboardMetrics {
  active: number
  on_time: number
  at_risk: number
  late: number
  completed: number
}

export interface CreateOrderPayload {
  client_id: string
  product: string
  specification?: string
  quantity: number
  unit: string
  promised_at: string
}

export const ordersApi = {
  list: () => api.get<{ data: Order[] }>('/api/v1/orders'),
  getByID: (id: string) => api.get<{ data: Order }>(`/api/v1/orders/${id}`),
  create: (body: CreateOrderPayload, idempotencyKey: string, csrfToken?: string) =>
    api.post<{ data: Order }>(
      '/api/v1/orders',
      body,
      csrfToken,
      { 'Idempotency-Key': idempotencyKey }
    ),
  updateStage: (orderId: string, position: number, status: StageStatus, csrfToken?: string) =>
    api.patch<{ data: OrderStage }>(
      `/api/v1/orders/${orderId}/stages/${position}`,
      { status },
      csrfToken
    ),
  createTrackingLink: (orderId: string, csrfToken?: string) =>
    api.post<{ data: { token: string } }>(`/api/v1/orders/${orderId}/tracking-link`, undefined, csrfToken),
  reopenStage: (orderId: string, position: number, reason: string, csrfToken?: string) =>
    api.post<{ data: OrderStage }>(
      `/api/v1/orders/${orderId}/stages/${position}/reopen`,
      { reason },
      csrfToken
    ),
  addStagePhoto: (orderId: string, position: number, photoData: string, caption?: string, csrfToken?: string) =>
    api.post<{ data: StagePhoto }>(
      `/api/v1/orders/${orderId}/stages/${position}/photos`,
      { photo_data: photoData, caption: caption || '' },
      csrfToken
    ),
  getDashboard: () => api.get<{ data: DashboardMetrics }>('/api/v1/dashboard'),
}
