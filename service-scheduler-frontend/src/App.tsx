import QuickBookingForm from './features/quick-booking/QuickBookingForm'
import './App.css'
import background from './assets/car-garage.png'

function App() {
  return (
    <div className="relative min-h-screen w-full justify-items-center">
      <div
        className="fixed inset-0 -z-10 bg-cover bg-center opacity-30"
        style={{ backgroundImage: `url(${background})` }}
        aria-hidden="true"
      />
      <QuickBookingForm />
    </div>
  )
}

export default App
