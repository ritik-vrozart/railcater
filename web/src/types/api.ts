export interface PaginatedMeta {
  page: number
  per_page: number
  total: number
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: PaginatedMeta
}

export interface Station {
  id: string
  code: string
  name: string
  city: string
  state: string
  created_at: string
}

export interface Vendor {
  id: string
  tenant_id: string
  name: string
  code: string
  phone?: string
  is_active: boolean
  is_approved: boolean
  created_at: string
  updated_at: string
}

export interface VendorTrain {
  train_id: string
  train_number: string
  train_name: string
  is_active: boolean
}

export interface VendorDetail extends Vendor {
  trains?: VendorTrain[]
  period_orders: number
  period_revenue_cents: number
  total_orders: number
  total_revenue_cents: number
  admin_email?: string
}

export interface AdminDashboard {
  date_from: string
  date_to: string
  total_pantries: number
  active_pantries: number
  period_orders: number
  period_revenue_cents: number
  total_orders: number
  total_revenue_cents: number
  pantries?: VendorDetail[]
}

export interface DailyMenuItem {
  id: string
  daily_menu_id: string
  menu_item_id: string
  menu_item_name?: string
  is_available: boolean
  stock_override?: number
  created_at: string
}

export interface DailyMenu {
  id: string
  vendor_id: string
  menu_date: string
  notes?: string
  items: DailyMenuItem[]
  created_at: string
  updated_at: string
}

export type FoodType = 'veg' | 'non_veg'

export interface MenuCategory {
  id: string
  vendor_id: string
  name: string
  description?: string
  food_type: FoodType
  sort_order: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export type PortionCode = 'quarter' | 'half' | 'full' | 'single'

export interface MenuItemPortion {
  id: string
  menu_item_id: string
  portion: PortionCode
  label: string
  price_cents: number
  stock_quantity: number
  is_active: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface MenuItem {
  id: string
  vendor_id: string
  category_id?: string
  category?: string
  food_type?: FoodType
  name: string
  description?: string
  image_url?: string
  price_cents: number
  is_veg: boolean
  is_active: boolean
  portions: MenuItemPortion[]
  total_stock: number
  created_at: string
  updated_at: string
}

export interface PortionFormRow {
  portion: PortionCode
  label: string
  priceRupees: string
  stockQuantity: string
  enabled: boolean
}

export interface OrderItem {
  id: string
  order_id: string
  product_id?: string
  menu_item_id?: string
  menu_portion_id?: string
  product_name?: string
  portion_label?: string
  portion?: string
  sku?: string
  quantity: number
  unit_price_cents: number
  line_total_cents: number
}

export type PaymentStatus = 'pending' | 'paid'
export type PaymentMethod = 'cash' | 'upi'

export interface Order {
  id: string
  tenant_id: string
  customer_id?: string
  customer_name?: string
  status: string
  source: string
  subtotal_cents: number
  total_cents: number
  notes?: string
  pnr?: string
  train_number?: string
  train_name?: string
  vendor_id?: string
  vendor_name?: string
  coach?: string
  berth?: string
  passenger_name?: string
  delivery_window_start?: string
  delivery_window_end?: string
  delivery_notified_at?: string
  expected_delivery_at?: string
  payment_status?: PaymentStatus
  payment_method?: PaymentMethod
  customer_phone?: string
  items?: OrderItem[]
  created_at: string
  updated_at: string
}
