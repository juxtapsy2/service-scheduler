import { apiClient } from '../../../api/client'

export type BookingRequest = {
  customer_id: string
  vehicle_id: string
  dealership_id: string
  service_type_id: string
  desired_start: string
}

export type BookingResponse = {
  appointment_id: string
  message: string
}

export async function createBooking(body: BookingRequest): Promise<BookingResponse> {
  return apiClient.post<BookingResponse>('/api/bookings', body)
}

export type QuickBookingRequest = {
  customer_first_name: string
  customer_last_name: string
  customer_email: string
  customer_phone?: string
  vehicle_vin: string
  vehicle_make: string
  vehicle_model: string
  vehicle_year: number
  dealership_id: string
  preferred_technician_id?: string
  service_type: string
  other_duration_minutes?: number
  desired_start: string
}

export async function createQuickBooking(body: QuickBookingRequest): Promise<BookingResponse> {
  return apiClient.post<BookingResponse>('/api/quick-booking', body)
}
