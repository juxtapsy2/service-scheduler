import { apiClient } from '../../../api/client'

export type Technician = {
  id: string
  first_name: string
  last_name: string
}

export async function listTechnicians(dealershipId: string): Promise<Technician[]> {
  const query = `?dealership_id=${encodeURIComponent(dealershipId)}`
  return apiClient.get<Technician[]>(`/api/technicians${query}`)
}
