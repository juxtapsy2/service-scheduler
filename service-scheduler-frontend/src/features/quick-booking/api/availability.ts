import { apiClient } from '../../../api/client'

export type AvailabilityRequest = {
  dealership_id: string
  preferred_technician_id?: string
  service_type: string
  other_duration_minutes?: number
  desired_start: string
}

export type AvailabilityResponse = {
  technician_available?: boolean
  technician_reason?: string
  bay_available: boolean
  bay_id?: string
  desired_end: string
}

export async function checkAvailability(body: AvailabilityRequest): Promise<AvailabilityResponse> {
  return apiClient.post<AvailabilityResponse>('/api/availability', body)
}
