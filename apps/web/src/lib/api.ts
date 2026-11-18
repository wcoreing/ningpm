const TOKEN_KEY = "ningpm_token"

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ""
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

type Envelope<T> = {
  ok: boolean
  data?: T
  error?: string
  version?: string
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || {})
  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json")
  }
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)

  const res = await fetch(path, { ...init, headers })
  const body = (await res.json().catch(() => null)) as Envelope<T> | null
  if (!res.ok || !body?.ok) {
    if (res.status === 401) {
      clearToken()
    }
    throw new ApiError(res.status, body?.error || "请求失败")
  }
  return body.data as T
}
