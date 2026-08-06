export const OptionsKey = {
  Dealerships: 'dealerships',
  Technicians: 'technicians',
  ServiceTypes: 'serviceTypes',
} as const

export type OptionsKey = (typeof OptionsKey)[keyof typeof OptionsKey]

export type FieldDef = {
  name: string
  label: string
  placeholder?: string
  type?: 'text' | 'datetime-local' | 'date' | 'time' | 'number' | 'email' | 'select'
  optionsKey?: OptionsKey
  required?: boolean
}
