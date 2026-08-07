import React, { useEffect, useState, useRef } from 'react'
import { listServiceTypes } from './api/serviceTypes'
import type { ServiceType } from './api/serviceTypes'
import { listTechnicians } from './api/technicians'
import { listDealerships } from './api/dealerships'
import { createQuickBooking } from './api/bookings'
import { localDatetimeToOffsetString } from './utils'
import { checkAvailability } from './api/availability'

import { validateField } from './validation'

import type { FieldDef } from './types'
import { OptionsKey } from './types'
import DynamicField from './components/DynamicField'

import { toast, ToastContainer } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'

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
  { name: 'service_type', label: 'Service type', required: true, type: 'select', optionsKey: OptionsKey.ServiceTypes },
  { name: 'preferred_technician_id', label: 'Preferred technician', type: 'select', optionsKey: OptionsKey.Technicians },
  { name: 'desired_start', label: 'Desired start', placeholder: '', type: 'datetime-local', required: true },
]

export default function QuickBookingForm({ fields }: Props) {
  const useFields = fields ?? defaultFields

  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([])
  const [dealerships, setDealerships] = useState<any[]>([])
  const [technicians, setTechnicians] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const formRef = useRef<any>(null)
  const suppressUntilRef = useRef<number>(0)
  const availSeqRef = useRef<number>(0)
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
      if (list.length > 0) setForm((f: any) => ({ ...f, service_type: list[0].id }))
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

    // compute a validation hint but do NOT block typing — regexes require a full
    // match, so rejecting intermediate values would prevent users from entering
    // emails/phones/VINs/years. Validation is enforced on submit (see submit()).
    const vError = validateField(name, v)
    setFieldErrors((fe: any) => ({ ...fe, [name]: vError }))

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

  const [availability, setAvailability] = useState<any | null>(null)
  const [, setAvailLoading] = useState(false)
  const [fieldErrors, setFieldErrors] = useState<Record<string,string | null>>({})

  // helper to run availability check (reads latest form from ref so WS-triggered checks use current values)
  const runAvailabilityCheck = async () => {
    // suppress availability checks briefly after a successful booking initiated from this client
    if (Date.now() < suppressUntilRef.current) return
    // avoid checking while this client is submitting a booking
    if (loading) return

    const cur = formRef.current ?? form
    if (!cur || !cur.desired_start || !cur.dealership_id) return
    const payload: any = {
      dealership_id: cur.dealership_id,
      desired_start: localDatetimeToOffsetString(cur.desired_start),
      service_type: cur.service_type || '',
    }
    if (cur.preferred_technician_id) payload.preferred_technician_id = cur.preferred_technician_id
    if (cur.service_type === '__other__') payload.other_duration_minutes = Number(cur.other_duration_minutes || 0)

    setAvailLoading(true)
    const seq = ++availSeqRef.current
    try {
      const resp = await checkAvailability(payload)
      // ignore if a newer check started
      if (availSeqRef.current !== seq) return
      setAvailability(resp)
    } catch (e) {
      console.error('availability check failed', e)
      // ignore if a newer check started
      if (availSeqRef.current !== seq) return
      setAvailability(null)
    } finally {
      if (availSeqRef.current === seq) setAvailLoading(false)
    }
  }

  // keep a ref of current form so WS handlers can read latest values
  useEffect(() => { formRef.current = form }, [form])

  // debounced availability check when time or technician/service changes
  useEffect(() => {
    const timer = setTimeout(() => {
      void runAvailabilityCheck()
    }, 400)
    return () => clearTimeout(timer)
  }, [form.desired_start, form.preferred_technician_id, form.service_type, form.other_duration_minutes, form.dealership_id])

  // websocket subscription to receive appointment events for the dealership; triggers availability re-check
  useEffect(() => {
    let ws: WebSocket | null = null
    if (!form.dealership_id) return
    try {
      const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
      const host = window.location.hostname
      const url = `${protocol}://${host}:8080/ws?dealership_id=${encodeURIComponent(form.dealership_id)}`
      ws = new WebSocket(url)
      ws.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data)
          if (data && data.type) {
            // appointment created/updated -> re-run availability check immediately
            // read latest form via ref inside runAvailabilityCheck
            void runAvailabilityCheck()
          }
        } catch (e) {
          // ignore
        }
      }
    } catch (e) {
      console.error('ws connect failed', e)
    }
    return () => { if (ws) ws.close() }
  }, [form.dealership_id])

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setResult(null)

    // enforce validation on submit (inline hints are non-blocking)
    const errors: Record<string, string | null> = {}
    for (const f of useFields) {
      errors[f.name] = validateField(f.name, form[f.name])
    }
    const invalid = useFields.find((f) => f.required && validateField(f.name, form[f.name]))
    if (invalid) {
      setFieldErrors(errors)
      toast.error('Please fix the invalid fields before booking.')
      return
    }

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
      toast.success('Booked: ' + resp.appointment_id)
      // suppress availability checks briefly to avoid the just-created appointment immediately flipping the UI
      suppressUntilRef.current = Date.now() + 1500
      // bump sequence to invalidate any in-flight availability responses
      availSeqRef.current++
      setAvailability(null)
    } catch (err: any) {
      const msg = err?.message || String(err)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="container py-4 w-full min-h-screen max-w-xl md:max-w-3xl flex flex-col items-center justify-center">
      <form onSubmit={submit} className="form-card w-full">
        <h2 className="form-title text-2xl">Service Scheduler</h2>
        <div className="row gx-3 gy-3">
          {useFields.map((f) => {
            const key = f.optionsKey
            let options: any[] = []
            if (key === OptionsKey.Dealerships) options = dealerships
            else if (key === OptionsKey.Technicians) options = [{ id: '', first_name: 'Any', last_name: 'available' }, ...technicians]
            else if (key === OptionsKey.ServiceTypes) options = [...serviceTypes, { id: '__other__', name: 'Other' }]

            return (
              <DynamicField
                key={f.name}
                field={f}
                value={form[f.name]}
                onChange={handleChange}
                options={options}
                error={fieldErrors[f.name]}
              />
            )
          })}

          {/* If user picked Other service type, show duration input (two-column) */}
          {form.service_type === '__other__' && (
            <div className="grid grid-cols-1 md:grid-cols-6 gap-y-2 items-center mb-3">
              <div className="md:col-span-2 md:text-right pr-4"><label className="text-sm font-semibold text-gray-800 form-label">Duration (minutes)</label></div>
              <div className="md:col-span-4">
                <input
                  name="other_duration_minutes"
                  placeholder="Duration (minutes)"
                  value={form.other_duration_minutes ?? 30}
                  onChange={handleChange}
                  className="w-full px-3 py-2 border rounded form-control text-sm text-gray-800"
                  type="number"
                  required
                />
                {fieldErrors['other_duration_minutes'] && <div className="text-sm text-red-700 mt-1">{fieldErrors['other_duration_minutes']}</div>}
              </div>
            </div>
          )}

          {/* Availability indicator row */}
          {!result && availability && (
            <div className="flex flex-col mt-4 text-wrap">
              {form.preferred_technician_id ? (
                availability.technician_available ? (
                  <span className="text-success">Technician available ✓</span>
                ) : (
                  <span className="text-danger">Error: {availability.technician_reason || 'not available'}</span>
                )
              ) : null}

              <span className="ml-3">
                {availability.bay_available ? (
                  <span className="text-success">Bay available ✓</span>
                ) : (
                  <span className="text-danger">No bay available</span>
                )}
              </span>
            </div>
          )}
        </div>

        <div className="flex justify-end align-items-center pt-4 gap-3">
          <button type="submit" disabled={loading} className="btn btn-primary rounded-md px-4 py-2 shadow-sm whitespace-nowrap">{loading ? 'Booking...' : 'Book'}</button>
        </div>
      </form>

      <ToastContainer position="top-right" autoClose={3500} />
    </div>
  )
}
