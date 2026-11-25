import { api } from '../session/api'
import type { Order } from './api'

export interface ProductionStage { name: string; client_description: string }
export interface ProductionSettings { stages: ProductionStage[]; version: number }
export const defaultStages: ProductionStage[] = ['CORTE', 'COSTURA', 'ACABADO', 'CONTROL DE CALIDAD', 'PLANCHADO', 'EMPAQUE', 'ENTREGA'].map(name => ({ name, client_description: '' }))
export const productionApi = {
  get: () => api.get<{ data: ProductionSettings }>('/api/v1/production-settings'),
  save: (body: ProductionSettings, csrf?: string) => api.patch<{ data: ProductionSettings }>('/api/v1/production-settings', body, csrf),
  operation: (id: string, body: { responsible_user_id: string | null; block_reason: string; version: number }, csrf?: string) => api.patch<{ data: Order }>(`/api/v1/orders/${id}/operation`, body, csrf),
}
