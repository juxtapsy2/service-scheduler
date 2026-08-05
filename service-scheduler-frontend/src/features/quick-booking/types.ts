export const enum OptionsKey {
  Dealerships = 'dealerships',
  Technicians = 'technicians',
  ServiceTypes = 'serviceTypes',
}

export type FieldDef = {
  name: string
  label: string
  placeholder?: string
  type?: 'text' | 'datetime-local' | 'date' | 'time' | 'number' | 'email' | 'select'
  optionsKey?: OptionsKey
  required?: boolean
}
