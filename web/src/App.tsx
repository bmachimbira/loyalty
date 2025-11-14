import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { ProtectedRoute } from './components/ProtectedRoute'
import { useAuth } from './contexts/AuthContext'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Customers from './pages/Customers'
import Rewards from './pages/Rewards'
import Rules from './pages/Rules'
import Campaigns from './pages/Campaigns'
import Budgets from './pages/Budgets'

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/login" element={<LoginRoute />} />

        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="customers" element={<Customers />} />
          <Route path="rewards" element={<Rewards />} />
          <Route path="rules" element={<Rules />} />
          <Route path="campaigns" element={<Campaigns />} />
          <Route path="budgets" element={<Budgets />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  )
}

// Redirect to dashboard if already logged in
function LoginRoute() {
  const { isAuthenticated } = useAuth()

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  return <Login />
}

export default App
