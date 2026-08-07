import { useEffect, useState, useRef } from 'react'
import { listServiceTypes } from '../api/serviceTypes'
import type { ServiceType } from '../api/serviceTypes'
import { listTechnicians } from '../api/technicians'
import { listDealerships } from '../api/dealerships'
import { createQuickBooking } from '../api/bookings'
import { localDatetimeToOffsetString } from '../utils'
import { checkAvailability } from '../api/availability'
import { validateField } from '../validation'
import { connectDealershipSocket } from '../websocket'
import type { FieldDef } from '../types'
import { OptionsKey } from '../types'
import { toast } from 'react-toastify'

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

export function useQuickBooking(fields?: FieldDef[]) {
  const useFields = fields ?? defaultFields

  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([])
  const [dealerships, setDealerships] = useState<any[]>([])
  const [technicians, setTechnicians] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const [availability, setAvailability] = useState<any | null>(null)
  const [, setAvailLoading] = useState(false)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string | null>>({})

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

  const formRef = useRef<any>(null)
  const suppressUntilRef = useRef<number>(0)
  const availSeqRef = useRef<number>(0)

  // load service types, dealerships and technicians for the default dealership
  useEffect(() => {
    let mounted = true

    listServiceTypes()
      .then((list) => {
        if (!mounted) return
        setServiceTypes(list)
        if (list.length > 0) setForm((f: any) => ({ ...f, service_type: list[0].id }))
      })
      .catch((e) => console.error('failed to load service types', e))

    listDealerships()
      .then((dlist) => {
        if (!mounted) return
        setDealerships(dlist)
        if (dlist.length > 0) {
          const first = dlist[0]
          setForm((f: any) => ({ ...f, dealership_id: first.id }))
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

  // reload technicians for a dealership/service combo and auto-pick the first one
  function reloadTechnicians(dealershipId: string, serviceType?: string) {
    const load = serviceType
      ? listTechnicians(dealershipId, serviceType)
      : listTechnicians(dealershipId)
    load
      .then((tlist) => {
        setTechnicians(tlist)
        setForm((f: any) => ({ ...f, preferred_technician_id: tlist.length > 0 ? tlist[0].id : '' }))
      })
      .catch((e) => console.error('failed to load technicians', e))
  }

  // generic change handler: computes a validation hint but never blocks typing
  // (regexes require a full match; validation is enforced on submit)
  function handleChange(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    const { name, value, type } = e.target as HTMLInputElement
    let v: any = value
    if (type === 'number') v = value === '' ? '' : Number(value)

    const vError = validateField(name, v)
    setFieldErrors((fe: any) => ({ ...fe, [name]: vError }))

    if (name === 'dealership_id') {
      setForm((f: any) => ({ ...f, dealership_id: value }))
      const svc = form.service_type
      reloadTechnicians(value, svc && svc !== '__other__' ? svc : undefined)
      return
    }

    if (name === 'service_type') {
      const svc = value
      setForm((f: any) => ({ ...f, service_type: svc }))
      if (form.dealership_id) reloadTechnicians(form.dealership_id, svc && svc !== '__other__' ? svc : undefined)
      return
    }

    setForm((f: any) => ({ ...f, [name]: v }))
  }

  // helper to run availability check (reads latest form from ref so WS-triggered checks use current values)
  async function runAvailabilityCheck() {
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

  // websocket subscription for dealership appointment events; triggers availability re-check
  useEffect(() => {
    if (!form.dealership_id) return
    return connectDealershipSocket(form.dealership_id, () => {
      void runAvailabilityCheck()
    })
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

  return {
    useFields,
    form,
    serviceTypes,
    dealerships,
    technicians,
    loading,
    result,
    availability,
    fieldErrors,
    handleChange,
    submit,
  }
}
