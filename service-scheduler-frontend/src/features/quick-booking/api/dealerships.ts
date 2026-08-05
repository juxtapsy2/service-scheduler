import { apiClient } from '../../../api/client'

export type Dealership = {
  id: string
  name: string
}

export async function listDealerships(): Promise<Dealership[]> {
  return apiClient.get<Dealership[]>('/api/dealerships')
}
