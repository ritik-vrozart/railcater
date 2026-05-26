import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Alert } from '../components/ui/Alert'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { useAuth } from '../context/AuthContext'
import { ApiError } from '../lib/api'

export function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await register({
        name,
        email,
        password,
        role: 'passenger',
        phone: phone.trim() || undefined,
      })
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Registration failed. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold text-white">Create account</h2>
      <p className="mt-2 text-sm text-slate-400">
        Passenger sign-up only. Pantry and department logins are issued by the train department head.
      </p>
      <p className="mt-1 text-sm text-slate-400">
        Already have an account?{' '}
        <Link to="/login" className="font-medium text-amber-400 hover:text-amber-300">
          Sign in
        </Link>
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-5">
        {error ? <Alert message={error} variant="dark" /> : null}

        <Input
          label="Full name"
          name="name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Rajesh Kumar"
        />

        <Input
          label="Email"
          name="email"
          type="email"
          autoComplete="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
        />

        <Input
          label="Phone (optional)"
          name="phone"
          type="tel"
          autoComplete="tel"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          placeholder="919876543210"
        />

        <Input
          label="Password"
          name="password"
          type="password"
          autoComplete="new-password"
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="At least 8 characters"
        />
        <p className="text-xs text-slate-500">Must include at least one letter and one number.</p>

        <Button type="submit" loading={loading}>
          Create account
        </Button>
      </form>
    </div>
  )
}
