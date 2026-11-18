import { renderAsync } from "docx-preview"
import * as XLSX from "xlsx"

export type PreviewKind = "image" | "pdf" | "docx" | "excel" | "none"

export function previewKind(name: string, contentType: string): PreviewKind {
  const n = (name || "").toLowerCase()
  const ct = (contentType || "").toLowerCase()
  if (ct.startsWith("image/") || /\.(png|jpe?g|webp|gif)$/.test(n)) return "image"
  if (ct === "application/pdf" || n.endsWith(".pdf")) return "pdf"
  if (
    ct.includes("wordprocessingml") ||
    ct === "application/msword" ||
    n.endsWith(".docx")
  ) {
    return "docx"
  }
  if (
    ct.includes("spreadsheetml") ||
    ct.includes("excel") ||
    /\.(xlsx|xls|csv)$/.test(n)
  ) {
    return "excel"
  }
  return "none"
}

export function fileKindLabel(kind: PreviewKind): string {
  switch (kind) {
    case "image":
      return "图"
    case "pdf":
      return "PDF"
    case "docx":
      return "DOC"
    case "excel":
      return "表"
    default:
      return "文"
  }
}

/** Render a DOCX blob into a container. */
export async function renderDocx(blob: Blob, container: HTMLElement) {
  container.innerHTML = ""
  await renderAsync(blob, container, undefined, {
    className: "np-docx",
    inWrapper: true,
    ignoreWidth: false,
    ignoreHeight: false,
    breakPages: true,
    renderHeaders: true,
    renderFooters: true,
  })
}

export type ExcelSheetView = {
  name: string
  html: string
}

/** Parse workbook into simple HTML tables (first few sheets). */
export async function parseExcel(blob: Blob): Promise<ExcelSheetView[]> {
  const buf = await blob.arrayBuffer()
  const wb = XLSX.read(buf, { type: "array" })
  return wb.SheetNames.slice(0, 8).map((name) => {
    const sheet = wb.Sheets[name]
    const html = XLSX.utils.sheet_to_html(sheet, { id: `sheet-${name}` })
    return { name, html }
  })
}
