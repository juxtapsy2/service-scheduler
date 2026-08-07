import type { FieldDef } from './types'
import { OptionsKey } from './types'
import DynamicField from './components/DynamicField'
import { useQuickBooking } from './useQuickBooking'

import { ToastContainer } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'

type Props = {
  fields?: FieldDef[]
}

export default function QuickBookingForm({ fields }: Props) {
  const {
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
  } = useQuickBooking(fields)

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
            <div className="flex flex-col items-end mt-4 text-wrap">
              {form.preferred_technician_id ? (
                availability.technician_available ? (
                  <span className="text-success">Technician available ✓</span>
                ) : (
                  <span className="text-danger">Technician: {availability.technician_reason || 'not available'}</span>
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
