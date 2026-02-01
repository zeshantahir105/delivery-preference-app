const BASE = (import.meta.env.VITE_API_URL as string) || ''

function getToken(): string | null {
  return localStorage.getItem('token')
}

export async function login(email: string, password: string): Promise<{ token: string }> {
  const res = await fetch(`${BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Login failed')
  return data as { token: string }
}

export async function me(): Promise<{ id: number; email: string }> {
  const token = getToken()
  if (!token) throw new Error('Not authenticated')
  const res = await fetch(`${BASE}/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Request failed')
  return data as { id: number; email: string }
}

export type OrderPreference = 'IN_STORE' | 'DELIVERY' | 'CURBSIDE'

export interface Order {
  id: number
  user_id: number
  preference: OrderPreference
  address?: string
  pickup_time?: string
  created_at: string
}

export async function getOrders(): Promise<Order[]> {
  const token = getToken()
  if (!token) throw new Error('Not authenticated')
  const res = await fetch(`${BASE}/orders`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Failed to load orders')
  return data as Order[]
}

export async function createOrder(body: {
  preference: OrderPreference
  address?: string
  pickup_time?: string
}): Promise<Order> {
  const token = getToken()
  if (!token) throw new Error('Not authenticated')
  const res = await fetch(`${BASE}/orders`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Create failed')
  return data as Order
}

export async function getOrder(id: number): Promise<Order> {
  const token = getToken()
  if (!token) throw new Error('Not authenticated')
  const res = await fetch(`${BASE}/orders/${id}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Not found')
  return data as Order
}

export async function updateOrder(
  id: number,
  body: { preference: OrderPreference; address?: string; pickup_time?: string }
): Promise<Order> {
  const token = getToken()
  if (!token) throw new Error('Not authenticated')
  const res = await fetch(`${BASE}/orders/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Update failed')
  return data as Order
}

/** AI-backed order summary (backend-proxied; OpenAI or Gemini when key set, else fallback). */
export async function getOrderSummary(orderId: number): Promise<{ summary: string; source?: string }> {
  const token = getToken()
  if (!token) throw new Error('Not authenticated')
  const res = await fetch(`${BASE}/orders/${orderId}/summary`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || 'Summary unavailable')
  return data as { summary: string; source?: string }
}
