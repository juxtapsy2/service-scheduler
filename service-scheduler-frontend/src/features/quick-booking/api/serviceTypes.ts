import { apiClient } from '../../../api/client'

export type ServiceType = { id: string; name: string; duration_minutes: number }

export async function listServiceTypes(): Promise<ServiceType[]> {
  return apiClient.get<ServiceType[]>('/api/service-types')
}
