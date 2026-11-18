/** Compress / downscale an image for avatar upload. Returns original if already small enough. */
export async function compressImage(
  file: File,
  opts: { maxEdge?: number; maxBytes?: number; quality?: number } = {},
): Promise<File> {
  const maxEdge = opts.maxEdge ?? 512
  const maxBytes = opts.maxBytes ?? 500 * 1024
  const quality = opts.quality ?? 0.82

  if (!file.type.startsWith("image/") || file.type === "image/gif") {
    return file
  }
  if (file.size <= maxBytes) {
    // still downscale huge dimensions that happen to be well-compressed
  }

  const bitmap = await createImageBitmap(file)
  try {
    const scale = Math.min(1, maxEdge / Math.max(bitmap.width, bitmap.height))
    const w = Math.max(1, Math.round(bitmap.width * scale))
    const h = Math.max(1, Math.round(bitmap.height * scale))

    if (scale === 1 && file.size <= maxBytes) {
      return file
    }

    const canvas = document.createElement("canvas")
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext("2d")
    if (!ctx) return file
    ctx.drawImage(bitmap, 0, 0, w, h)

    const preferWebp = supportsWebp()
    let q = quality
    let blob: Blob | null = null
    for (let i = 0; i < 6; i++) {
      blob = await canvasToBlob(canvas, preferWebp ? "image/webp" : "image/jpeg", q)
      if (!blob || blob.size <= maxBytes) break
      q = Math.max(0.45, q - 0.12)
    }
    if (!blob) return file

    // keep original if compression did not help
    if (blob.size >= file.size && scale === 1) return file

    const ext = preferWebp ? "webp" : "jpg"
    const base = file.name.replace(/\.[^.]+$/, "") || "avatar"
    return new File([blob], `${base}.${ext}`, {
      type: blob.type,
      lastModified: Date.now(),
    })
  } finally {
    bitmap.close()
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number) {
  return new Promise<Blob | null>((resolve) => {
    canvas.toBlob((b) => resolve(b), type, quality)
  })
}

let webpOk: boolean | null = null
function supportsWebp() {
  if (webpOk != null) return webpOk
  try {
    webpOk = document.createElement("canvas").toDataURL("image/webp").startsWith("data:image/webp")
  } catch {
    webpOk = false
  }
  return webpOk
}
