import { ApiError, getToken, clearToken } from "./api"

type Envelope<T> = {
  ok: boolean
  data?: T
  error?: string
}

export type ResourceMeta = {
  id: string
  originalName: string
  contentType: string
  sizeBytes: number
  uploaderId: string
  createdAt: string
  url?: string
}

/** Upload a file to the shared resource service (multipart; do not set JSON Content-Type). */
export async function uploadResource(file: File): Promise<ResourceMeta> {
  const body = new FormData()
  body.append("file", file)
  const headers = new Headers()
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)

  const res = await fetch("/api/resources", { method: "POST", headers, body })
  const json = (await res.json().catch(() => null)) as Envelope<ResourceMeta> | null
  if (!res.ok || !json?.ok || !json.data) {
    if (res.status === 401) clearToken()
    throw new ApiError(res.status, json?.error || "上传失败")
  }
  return json.data
}

/** Fetch an authenticated resource as a Blob. */
export async function fetchResourceBlob(url: string): Promise<Blob> {
  const headers = new Headers()
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const res = await fetch(url, { headers })
  if (!res.ok) {
    if (res.status === 401) clearToken()
    throw new ApiError(res.status, "加载失败")
  }
  return res.blob()
}

/** Download an authenticated resource as a browser download. */
export async function downloadResource(url: string, filename: string) {
  const blob = await fetchResourceBlob(url)
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = objectUrl
  a.download = filename || "download"
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objectUrl)
}

export function isImageContentType(ct: string) {
  return /^image\/(png|jpe?g|webp|gif)$/i.test(ct || "")
}

export { previewKind, fileKindLabel } from "./preview"

export function formatBytes(n: number) {
  if (!Number.isFinite(n) || n < 0) return "-"
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
