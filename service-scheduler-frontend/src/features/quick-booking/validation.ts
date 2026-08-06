// Re-usable validation regexes and helpers for quick-booking form
export const REGEX = {
  NAME: /^[\p{L} .'-]{1,100}$/u, // Unicode letters, spaces, dots, apostrophes, hyphens
  EMAIL: /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/,
  PHONE: /^[0-9+() \-]{6,20}$/, // digits and common separators
  VIN: /^[A-HJ-NPR-Z0-9]{11,17}$/i, // common VIN pattern (no I,O,Q)
  YEAR: /^(19|20)\d{2}$/, // 1900-2099
  UUID: /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/, // simple UUID v4-ish
  DURATION: /^\d{1,4}$/, // duration minutes up to 9999
  DATETIME_LOCAL: /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/,
}

export function validateField(name: string, value: any): string | null {
  if (value === undefined || value === null || value === '') return null
  switch (name) {
    case 'customer_first_name':
    case 'customer_last_name':
      return REGEX.NAME.test(String(value)) ? null : 'Invalid name'
    case 'customer_email':
      return REGEX.EMAIL.test(String(value)) ? null : 'Invalid email'
    case 'customer_phone':
      return REGEX.PHONE.test(String(value)) ? null : 'Invalid phone'
    case 'vehicle_vin':
      return REGEX.VIN.test(String(value)) ? null : 'Invalid VIN'
    case 'vehicle_year':
      return REGEX.YEAR.test(String(value)) ? null : 'Invalid year'
    case 'dealership_id':
    case 'preferred_technician_id':
      return REGEX.UUID.test(String(value)) ? null : 'Invalid selection'
    case 'other_duration_minutes':
      return REGEX.DURATION.test(String(value)) ? null : 'Invalid duration'
    case 'desired_start':
      return REGEX.DATETIME_LOCAL.test(String(value)) ? null : 'Invalid datetime'
    default:
      return null
  }
}
