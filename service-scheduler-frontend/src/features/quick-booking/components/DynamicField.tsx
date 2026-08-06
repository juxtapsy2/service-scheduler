import React from 'react'
import type { FieldDef, OptionsKey } from './types'

type Props = {
  field: FieldDef
  value: any
  onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => void
  options?: any[]
  error?: string | null
}

export default function DynamicField({ field, value, onChange, options = [], error }: Props) {
  const { name, label, placeholder, type = 'text', required } = field

  return (
    <div className="grid grid-cols-1 md:grid-cols-6 gap-y-2 items-center mb-3">
      <div className="md:col-span-2 md:text-right pr-4">
        <label className="text-sm font-semibold text-gray-800 form-label">{label}</label>
      </div>
      <div className="md:col-span-4">
        {type === 'select' ? (
          <select name={name} value={value ?? ''} onChange={onChange} className="w-full px-3 py-2 border rounded form-select" required={required}>
            {/* allow caller to include empty option for technicians */}
            {options.length === 0 && <option value="">(no options)</option>}
            {options.map((o: any) => {
              // try to be flexible about option shape
              if (o.id && o.name) return <option key={o.id} value={o.id}>{o.name}</option>
              if (o.id && o.first_name) return <option key={o.id} value={o.id}>{o.first_name} {o.last_name}</option>
              if (typeof o === 'string') return <option key={o} value={o}>{o}</option>
              return <option key={o.id ?? (o.value || Math.random())} value={o.id ?? o.value}>{o.name ?? o.label ?? o.id}</option>
            })}
          </select>
        ) : (
          <input
            name={name}
            placeholder={placeholder || label}
            value={value ?? ''}
            onChange={onChange}
            className={`w-full px-3 py-2 border rounded form-control`}
            required={required}
            type={type}
          />
        )}
        {error && <div className="text-sm text-red-700 mt-1">{error}</div>}
      </div>
    </div>
  )
}
