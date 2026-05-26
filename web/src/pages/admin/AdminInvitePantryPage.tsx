import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Alert } from '../../components/ui/Alert'
import { ButtonLight } from '../../components/ui/ButtonLight'
import { FormField, TextInput } from '../../components/ui/FormField'
import { PageHeader } from '../../components/ui/PageHeader'
import { useAuth } from '../../context/AuthContext'
import { apiRequest, ApiError } from '../../lib/api'

function SectionTitle({ title, description }: { title: string; description?: string }) {
  return (
    <div className="mb-5">
      <h3 className="text-base font-semibold text-gray-900">{title}</h3>
      {description ? <p className="mt-1 text-sm text-gray-500">{description}</p> : null}
    </div>
  )
}

export function AdminInvitePantryPage() {
  const { token } = useAuth()
  const navigate = useNavigate()
  const [pantryName, setPantryName] = useState('')
  const [trainNumber, setTrainNumber] = useState('')
  const [trainName, setTrainName] = useState('')
  const [adminName, setAdminName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [phone, setPhone] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!token) return
    setError('')
    setLoading(true)
    try {
      await apiRequest(
        '/api/v1/admin/pantries/invite',
        {
          method: 'POST',
          body: JSON.stringify({
            pantry_name: pantryName,
            train_number: trainNumber,
            train_name: trainName,
            admin_name: adminName || pantryName,
            email,
            password,
            phone: phone.trim() || undefined,
          }),
        },
        token,
      )
      navigate('/admin/pantries', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Invite failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full">
      <PageHeader
        title="Invite pantry"
        description="Link a new pantry to a train and create manager login"
        actions={
          <Link
            to="/admin/pantries"
            className="text-sm font-medium text-amber-600 hover:text-amber-500"
          >
            ← Back to pantries
          </Link>
        }
      />

      
      {error ? (
        <div className="mb-6">
          <Alert message={error} />
        </div>
      ) : null}

      <form onSubmit={handleSubmit} autoComplete="off" className="w-full">
        <div className="grid w-full gap-10 lg:grid-cols-2 lg:gap-12">
          <section>
            <SectionTitle
              title="Pantry & train"
              description="Which train this pantry serves"
            />
            <div className="space-y-4">
              <FormField label="Pantry name">
                <TextInput
                  value={pantryName}
                  onChange={(e) => setPantryName(e.target.value)}
                  placeholder="e.g. Rajdhani Pantry Car 1"
                  autoComplete="off"
                  required
                />
              </FormField>
              <div className="grid gap-4 sm:grid-cols-2">
                <FormField label="Train number">
                  <TextInput
                    value={trainNumber}
                    onChange={(e) => setTrainNumber(e.target.value)}
                    placeholder="12951"
                    inputMode="numeric"
                    autoComplete="off"
                    required
                  />
                </FormField>
                <FormField label="Train name">
                  <TextInput
                    value={trainName}
                    onChange={(e) => setTrainName(e.target.value)}
                    placeholder="Rajdhani Express"
                    autoComplete="off"
                  />
                </FormField>
              </div>
            </div>
          </section>

          <section>
            <SectionTitle
              title="Manager login"
              description="Credentials for the pantry manager (not your admin account)"
            />
            <div className="space-y-4">
              <FormField label="Manager name" hint="Optional — defaults to pantry name">
                <TextInput
                  value={adminName}
                  onChange={(e) => setAdminName(e.target.value)}
                  placeholder="e.g. Ramesh Kumar"
                  autoComplete="off"
                />
              </FormField>
              <FormField label="Login email">
                <TextInput
                  type="email"
                  name="pantry-manager-email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="manager@pantry.example.com"
                  autoComplete="off"
                  required
                />
              </FormField>
              <FormField label="Password" hint="Min 8 characters, with letters and numbers">
                <TextInput
                  type="password"
                  name="pantry-manager-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Create a secure password"
                  autoComplete="new-password"
                  required
                />
              </FormField>
              <FormField label="Phone (optional)">
                <TextInput
                  type="tel"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  placeholder="919876543210"
                  autoComplete="off"
                />
              </FormField>
            </div>
          </section>
        </div>

        <div className="mt-10 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <ButtonLight
            type="button"
            variant="ghost"
            onClick={() => navigate('/admin/pantries')}
            disabled={loading}
          >
            Cancel
          </ButtonLight>
          <ButtonLight
            type="submit"
            disabled={loading}
            className="min-w-[220px] bg-amber-600 hover:bg-amber-500 focus:ring-amber-500"
          >
            {loading ? 'Creating pantry…' : 'Create pantry & login'}
          </ButtonLight>
        </div>
      </form>
    </div>
  )
}
