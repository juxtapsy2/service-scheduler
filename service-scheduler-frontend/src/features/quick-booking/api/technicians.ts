import { apiClient } from '../../../api/client'

export type Technician = {
  id: string
  first_name: string
  last_name: string
}

export async function listTechnicians(dealershipId: string, serviceType?: string): Promise<Technician[]> {
  let query = `?dealership_id=${encodeURIComponent(dealershipId)}`
  if (serviceType) {
    query += `&service_type=${encodeURIComponent(serviceType)}`
  }
  return apiClient.get<Technician[]>(`/api/technicians${query}`)
}
