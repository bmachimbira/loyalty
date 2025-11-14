import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import './App.css'

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-background">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/customers" element={<div>Customers Page</div>} />
          <Route path="/rewards" element={<div>Rewards Page</div>} />
          <Route path="/rules" element={<div>Rules Page</div>} />
          <Route path="/campaigns" element={<div>Campaigns Page</div>} />
          <Route path="/budgets" element={<div>Budgets Page</div>} />
        </Routes>
      </div>
    </Router>
  )
}

export default App
