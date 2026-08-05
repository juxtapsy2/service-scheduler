import React, { useEffect, useState } from 'react'
import { listServiceTypes } from './api/serviceTypes'
import type { ServiceType } from './api/serviceTypes'
import { createQuickBooking } from './api/bookings'

export type FieldDef = {
  name: string
  label: string
  placeholder?: string
  type?: 'text' | 'datetime-local' | 'date' | 'time' | 'number' | 'email'
  required?: boolean
}

type Props = {
  fields?: FieldDef[]
}

const defaultFields: FieldDef[] = [
  { name: 'customer_first_name', label: 'First name', required: true },
  { name: 'customer_last_name', label: 'Last name', required: true },
  { name: 'customer_email', label: 'Email', type: 'email', required: true },
  { name: 'customer_phone', label: 'Phone' },
  { name: 'vehicle_vin', label: 'Vehicle VIN', required: true },
  { name: 'vehicle_make', label: 'Make', required: true },
  { name: 'vehicle_model', label: 'Model', required: true },
  { name: 'vehicle_year', label: 'Year', type: 'number', required: true },
  { name: 'desired_start', label: 'Desired start (ISO8601)', placeholder: '2026-08-10T09:00:00Z', required: true },
]

export default function QuickBookingForm({ fields }: Props) {
  const useFields = fields ?? defaultFields

  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([])
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState<any>({
    customer_first_name: '',
    customer_last_name: '',
    customer_email: '',
    customer_phone: '',
    vehicle_vin: '',
    vehicle_make: '',
    vehicle_model: '',
    vehicle_year: new Date().getFullYear(),
    dealership_id: '11111111-1111-1111-1111-111111111111',
    service_type: '',
    desired_start: '',
  })

  useEffect(() => {
    let mounted = true
    listServiceTypes()
      .then((list) => {
        if (!mounted) return
        setServiceTypes(list)
        if (list.length > 0) setForm((f: any) => ({ ...f, service_type: list[0].name }))
      })
      .catch((e) => {
        console.error('failed to load service types', e)
      })
    return () => { mounted = false }
  }, [])

  function handleChange(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    const { name, value } = e.target
    setForm((f: any) => ({ ...f, [name]: value }))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setResult(null)
    setLoading(true)
    try {
      const payload = {
        ...form,
        vehicle_year: Number(form.vehicle_year),
      }
      const resp = await createQuickBooking(payload)
      setResult(resp.appointment_id)
    } catch (err: any) {
      setError(err.message || String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="p-6 max-w-xl mx-auto">
      <h2 className="text-2xl font-semibold mb-4">Quick Booking</h2>
      <form onSubmit={submit} className="space-y-3 bg-white p-4 rounded shadow">
        <div className="grid grid-cols-2 gap-3">
          {useFields.map((f) => (
            <input
              key={f.name}
              name={f.name}
              placeholder={f.placeholder || f.label}
              value={form[f.name] ?? ''}
              onChange={handleChange}
              className={`p-2 border rounded ${f.type === 'number' ? '' : ''}`}
              required={f.required}
              type={(f.type as any) || 'text'}
            />
          ))}
        </div>

        <div>
          <label className="block text-sm mb-1">Service type</label>
          <select name="service_type" value={form.service_type} onChange={handleChange} className="w-full p-2 border rounded">
            {serviceTypes.map((s) => <option key={s.id} value={s.name}>{s.name}</option>)}
          </select>
        </div>

        <div>
          <label className="block text-sm mb-1">Desired start (ISO8601)</label>
          <input name="desired_start" placeholder="2026-08-10T09:00:00Z" value={form.desired_start} onChange={handleChange} className="w-full p-2 border rounded" required />
        </div>

        <div className="flex items-center justify-between">
          <button type="submit" disabled={loading} className="bg-indigo-600 text-white px-4 py-2 rounded">{loading ? 'Booking...' : 'Book'}</button>
          {result && <span className="text-green-600">Booked: {result}</span>}
          {error && <span className="text-red-600">{error}</span>}
        </div>
      </form>
    </div>
  )
}
