import React, { useEffect, useState } from 'react'
import { listServiceTypes } from './api/serviceTypes'
import type { ServiceType } from './api/serviceTypes'
import { listTechnicians } from './api/technicians'
import { listDealerships } from './api/dealerships'
import { createQuickBooking } from './api/bookings'
import { localDatetimeToOffsetString } from './utils'

import type { FieldDef } from './types'
import { OptionsKey } from './types'

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
  { name: 'dealership_id', label: 'Dealership', required: true, type: 'select', optionsKey: OptionsKey.Dealerships },
  { name: 'preferred_technician_id', label: 'Preferred technician', type: 'select', optionsKey: OptionsKey.Technicians },
  { name: 'service_type', label: 'Service type', required: true, type: 'select', optionsKey: OptionsKey.ServiceTypes },
  { name: 'desired_start', label: 'Desired start', placeholder: '', type: 'datetime-local', required: true },
]

export default function QuickBookingForm({ fields }: Props) {
  const useFields = fields ?? defaultFields

  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([])
  const [dealerships, setDealerships] = useState<any[]>([])
  const [technicians, setTechnicians] = useState<any[]>([])
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
    preferred_technician_id: '',
    service_type: '',
    other_duration_minutes: 30,
    desired_start: '',
  })

  useEffect(() => {
    let mounted = true

    // load service types and dealerships in parallel
    listServiceTypes()
      .then((list) => {
        if (!mounted) return
        setServiceTypes(list)
        if (list.length > 0) setForm((f: any) => ({ ...f, service_type: list[0].name }))
      })
      .catch((e) => console.error('failed to load service types', e))

    // dealerships
    listDealerships()
      .then((dlist) => {
        if (!mounted) return
        setDealerships(dlist)
        if (dlist.length > 0) {
          const first = dlist[0]
          setForm((f: any) => ({ ...f, dealership_id: first.id }))
          // load technicians for first dealership
          listTechnicians(first.id)
            .then((tlist) => {
              if (!mounted) return
              setTechnicians(tlist)
              if (tlist.length > 0) setForm((f: any) => ({ ...f, preferred_technician_id: tlist[0].id }))
            })
            .catch((e) => console.error('failed to load technicians', e))
        }
      })
      .catch((e) => console.error('failed to load dealerships', e))

    return () => { mounted = false }
  }, [])

  function handleChange(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    const { name, value, type } = e.target as HTMLInputElement
    let v: any = value
    if (type === 'number') v = value === '' ? '' : Number(value)

    // if dealership changes, reload technicians (respect currently-selected service type)
    if (name === 'dealership_id') {
      const dealerId = value
      // optimistic update
      setForm((f: any) => ({ ...f, dealership_id: dealerId }))
      const svc = form.service_type
      if (svc && svc !== '__other__') {
        listTechnicians(dealerId, svc)
          .then((tlist) => {
            setTechnicians(tlist)
            setForm((f: any) => ({ ...f, preferred_technician_id: tlist.length > 0 ? tlist[0].id : '' }))
          })
          .catch((e) => console.error('failed to load technicians', e))
      } else {
        listTechnicians(dealerId)
          .then((tlist) => {
            setTechnicians(tlist)
            setForm((f: any) => ({ ...f, preferred_technician_id: tlist.length > 0 ? tlist[0].id : '' }))
          })
          .catch((e) => console.error('failed to load technicians', e))
      }
      return
    }

    // if service type changes, reload technicians filtered by service type (unless Other selected)
    if (name === 'service_type') {
      const svc = value
      setForm((f: any) => ({ ...f, service_type: svc }))
      const dealerId = form.dealership_id
      if (!dealerId) return
      if (svc && svc !== '__other__') {
        listTechnicians(dealerId, svc)
          .then((tlist) => {
            setTechnicians(tlist)
            setForm((f: any) => ({ ...f, preferred_technician_id: tlist.length > 0 ? tlist[0].id : '' }))
          })
          .catch((e) => console.error('failed to load technicians', e))
      } else {
        // Other: show all technicians (no qualification)
        listTechnicians(dealerId)
          .then((tlist) => {
            setTechnicians(tlist)
            setForm((f: any) => ({ ...f, preferred_technician_id: tlist.length > 0 ? tlist[0].id : '' }))
          })
          .catch((e) => console.error('failed to load technicians', e))
      }
      return
    }

    // store datetime-local value as-is (e.g. "2026-08-10T09:00")
    setForm((f: any) => ({ ...f, [name]: v }))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setResult(null)
    setLoading(true)
    try {
      // convert datetime-local to timestamp with local offset so backend receives user's local hour
      const desired = form.desired_start
      const desiredISO = desired ? localDatetimeToOffsetString(desired) : ''

      const payload = {
        ...form,
        vehicle_year: Number(form.vehicle_year),
        desired_start: desiredISO,
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
          {useFields.map((f) => {
            if (f.type === 'select') {
              const key = f.optionsKey
              const options = key === OptionsKey.Dealerships ? dealerships : key === OptionsKey.Technicians ? technicians : key === OptionsKey.ServiceTypes ? serviceTypes : []

              return (
                <select key={f.name} name={f.name} value={form[f.name] ?? ''} onChange={handleChange} className="p-2 border rounded" required={f.required}>
                  {key === OptionsKey.Technicians && <option value="">(Any available)</option>}
                  {options.map((o: any) => {
                    switch (key) {
                      case OptionsKey.ServiceTypes:
                        return <option key={o.id} value={o.name}>{o.name}</option>
                      case OptionsKey.Dealerships:
                        return <option key={o.id} value={o.id}>{o.name}</option>
                      case OptionsKey.Technicians:
                        return <option key={o.id} value={o.id}>{o.first_name} {o.last_name}</option>
                      default:
                        return <option key={o.id} value={o.id}>{o.name ?? o.id}</option>
                    }
                  })}
                  {/* allow ad-hoc Other service type which skips qualification */}
                  {key === OptionsKey.ServiceTypes && <option key="__other__" value="__other__">Other</option>}
                </select>
              )
            }

            return (
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
            )
          })}
          {/* If user picked Other service type, show duration input */}
          {form.service_type === '__other__' && (
            <input
              name="other_duration_minutes"
              placeholder="Duration (minutes)"
              value={form.other_duration_minutes ?? 30}
              onChange={handleChange}
              className="p-2 border rounded"
              type="number"
              required
              />
          )}
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
