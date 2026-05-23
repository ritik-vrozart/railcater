export type UserRole =
  | 'passenger'
  | 'vendor_admin'
  | 'kitchen_staff'
  | 'delivery_agent'
  | 'operations_manager'
  | 'super_admin'

export interface User {
  id: string
  tenant_id: string
  vendor_id?: string
  name: string
  email: string
  phone?: string
  role: UserRole
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface RegisterPayload {
  name: string
  email: string
  phone?: string
  password: string
  role?: UserRole
}

export interface LoginPayload {
  email: string
  password: string
}
