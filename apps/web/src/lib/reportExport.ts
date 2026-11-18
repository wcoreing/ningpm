import ExcelJS from "exceljs"

/** Web token colors (tokens.css) */
const C = {
  text: "1E293B",
  secondary: "475569",
  muted: "64748B",
  border: "C5D6EA",
  borderStrong: "9EB6D4",
  elevated: "FFFFFF",
  sunken: "D9E6F4",
  page: "E8F0F8",
  accent: "409EFF",
  accentInk: "2B6CB0",
  warning: "E6A23C",
  success: "67C23A",
  todoBar: "A8B3C2",
} as const

const STATUS_FILL: Record<string, string> = {
  todo: C.todoBar,
  doing: C.accent,
  review: C.warning,
  done: C.success,
}

export type ReportSummary = {
  total: number
  done: number
  doing: number
  review: number
  todo: number
  rate: number
  roots: number
  rootDone: number
  children: number
  childDone: number
  childRate: number
}

export type ReportStatRow = {
  label: string
  total: number
  done: number
  doing: number
  todo: number
  review: number
}

export type ReportExportInput = {
  projectKey: string
  projectName: string
  summary: ReportSummary
  byStatus: { status: string; label: string; count: number }[]
  byType: ReportStatRow[]
  byCycle: ReportStatRow[]
  byModule: ReportStatRow[]
}

function rate(done: number, total: number) {
  if (!total) return "0%"
  return `${Math.round((done / total) * 100)}%`
}

function stamp() {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}`
}

function thinBorder(): Partial<ExcelJS.Borders> {
  const edge: Partial<ExcelJS.Border> = { style: "thin", color: { argb: `FF${C.border}` } }
  return { top: edge, left: edge, bottom: edge, right: edge }
}

function fillSolid(hex: string): ExcelJS.Fill {
  return { type: "pattern", pattern: "solid", fgColor: { argb: `FF${hex}` } }
}

function styleRange(
  ws: ExcelJS.Worksheet,
  r1: number,
  c1: number,
  r2: number,
  c2: number,
  apply: (cell: ExcelJS.Cell) => void,
) {
  for (let r = r1; r <= r2; r++) {
    for (let c = c1; c <= c2; c++) apply(ws.getCell(r, c))
  }
}

/** Draw status bars matching web `.status-bars` */
async function renderStatusChart(
  rows: { status: string; label: string; count: number }[],
  total: number,
): Promise<ArrayBuffer> {
  const dpr = 2
  const width = 720
  const rowH = 36
  const padX = 16
  const padY = 12
  const labelW = 72
  const countW = 88
  const height = padY * 2 + Math.max(rows.length, 1) * rowH
  const canvas = document.createElement("canvas")
  canvas.width = width * dpr
  canvas.height = height * dpr
  const ctx = canvas.getContext("2d")
  if (!ctx) throw new Error("无法绘制图表")
  ctx.scale(dpr, dpr)

  ctx.fillStyle = `#${C.elevated}`
  ctx.fillRect(0, 0, width, height)

  rows.forEach((row, i) => {
    const y = padY + i * rowH
    const mid = y + rowH / 2
    ctx.fillStyle = `#${C.secondary}`
    ctx.font = "600 13px 'Noto Sans SC', 'PingFang SC', sans-serif"
    ctx.textAlign = "left"
    ctx.textBaseline = "middle"
    ctx.fillText(row.label, padX, mid)

    const trackX = padX + labelW
    const trackW = width - padX * 2 - labelW - countW
    const trackY = mid - 4
    const trackH = 8
    const radius = 4

    ctx.fillStyle = `#${C.sunken}`
    roundRect(ctx, trackX, trackY, trackW, trackH, radius)
    ctx.fill()

    const pct = total ? row.count / total : 0
    const fillW = Math.max(0, Math.round(trackW * pct))
    if (fillW > 0) {
      ctx.fillStyle = `#${STATUS_FILL[row.status] || C.accent}`
      roundRect(ctx, trackX, trackY, Math.max(fillW, fillW > 0 ? 6 : 0), trackH, radius)
      ctx.fill()
    }

    ctx.fillStyle = `#${C.muted}`
    ctx.font = "12px 'Noto Sans SC', 'PingFang SC', sans-serif"
    ctx.textAlign = "right"
    const pctLabel = total ? Math.round((row.count / total) * 100) : 0
    ctx.fillText(`${row.count} · ${pctLabel}%`, width - padX, mid)
  })

  return canvasToPng(canvas)
}

/** Mini horizontal bars for type / cycle tables */
async function renderMiniBars(
  rows: ReportStatRow[],
  title: string,
): Promise<ArrayBuffer> {
  const dpr = 2
  const width = 340
  const rowH = 28
  const padX = 12
  const padY = 28
  const labelW = 88
  const countW = 52
  const height = padY + 8 + Math.max(rows.length, 1) * rowH
  const canvas = document.createElement("canvas")
  canvas.width = width * dpr
  canvas.height = height * dpr
  const ctx = canvas.getContext("2d")
  if (!ctx) throw new Error("无法绘制图表")
  ctx.scale(dpr, dpr)

  ctx.fillStyle = `#${C.elevated}`
  ctx.fillRect(0, 0, width, height)
  ctx.fillStyle = `#${C.text}`
  ctx.font = "700 13px 'Noto Sans SC', 'PingFang SC', sans-serif"
  ctx.textAlign = "left"
  ctx.textBaseline = "top"
  ctx.fillText(title, padX, 8)

  const max = Math.max(1, ...rows.map((r) => r.total))
  rows.forEach((row, i) => {
    const y = padY + i * rowH
    const mid = y + rowH / 2
    ctx.fillStyle = `#${C.secondary}`
    ctx.font = "12px 'Noto Sans SC', 'PingFang SC', sans-serif"
    ctx.textAlign = "left"
    ctx.textBaseline = "middle"
    const label = row.label.length > 6 ? `${row.label.slice(0, 6)}…` : row.label
    ctx.fillText(label, padX, mid)

    const trackX = padX + labelW
    const trackW = width - padX * 2 - labelW - countW
    const trackY = mid - 4
    ctx.fillStyle = `#${C.sunken}`
    roundRect(ctx, trackX, trackY, trackW, 8, 4)
    ctx.fill()

    const fillW = Math.round((row.total / max) * trackW)
    if (fillW > 0) {
      // stacked: done / review / doing / todo
      let x = trackX
      const parts: [number, string][] = [
        [row.done, C.success],
        [row.review, C.warning],
        [row.doing, C.accent],
        [row.todo, C.todoBar],
      ]
      for (const [n, color] of parts) {
        if (!n || !row.total) continue
        const w = Math.max(2, Math.round((n / row.total) * fillW))
        ctx.fillStyle = `#${color}`
        roundRect(ctx, x, trackY, Math.min(w, trackX + fillW - x), 8, 2)
        ctx.fill()
        x += w
        if (x >= trackX + fillW) break
      }
    }

    ctx.fillStyle = `#${C.muted}`
    ctx.textAlign = "right"
    ctx.fillText(String(row.total), width - padX, mid)
  })

  return canvasToPng(canvas)
}

function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
) {
  const rr = Math.min(r, w / 2, h / 2)
  ctx.beginPath()
  ctx.moveTo(x + rr, y)
  ctx.arcTo(x + w, y, x + w, y + h, rr)
  ctx.arcTo(x + w, y + h, x, y + h, rr)
  ctx.arcTo(x, y + h, x, y, rr)
  ctx.arcTo(x, y, x + w, y, rr)
  ctx.closePath()
}

function canvasToPng(canvas: HTMLCanvasElement): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(async (blob) => {
      if (!blob) {
        reject(new Error("图表导出失败"))
        return
      }
      resolve(await blob.arrayBuffer())
    }, "image/png")
  })
}

function paintCard(
  ws: ExcelJS.Worksheet,
  r1: number,
  c1: number,
  r2: number,
  c2: number,
  label: string,
  value: string,
  hint: string,
) {
  ws.mergeCells(r1, c1, r1, c2)
  ws.mergeCells(r1 + 1, c1, r1 + 1, c2)
  ws.mergeCells(r1 + 2, c1, r2, c2)
  const labelCell = ws.getCell(r1, c1)
  labelCell.value = label
  labelCell.font = { name: "微软雅黑", size: 10, bold: true, color: { argb: `FF${C.muted}` } }
  labelCell.alignment = { vertical: "middle", horizontal: "left" }

  const valueCell = ws.getCell(r1 + 1, c1)
  valueCell.value = value
  valueCell.font = { name: "微软雅黑", size: 18, bold: true, color: { argb: `FF${C.text}` } }
  valueCell.alignment = { vertical: "middle", horizontal: "left" }

  const hintCell = ws.getCell(r1 + 2, c1)
  hintCell.value = hint
  hintCell.font = { name: "微软雅黑", size: 9, color: { argb: `FF${C.secondary}` } }
  hintCell.alignment = { vertical: "middle", horizontal: "left", wrapText: true }

  styleRange(ws, r1, c1, r2, c2, (cell) => {
    cell.fill = fillSolid(C.elevated)
    cell.border = thinBorder()
  })
}

function writeTable(
  ws: ExcelJS.Worksheet,
  startRow: number,
  startCol: number,
  title: string,
  headers: string[],
  rows: (string | number)[][],
) {
  const endCol = startCol + headers.length - 1
  ws.mergeCells(startRow, startCol, startRow, endCol)
  const titleCell = ws.getCell(startRow, startCol)
  titleCell.value = title
  titleCell.font = { name: "微软雅黑", size: 12, bold: true, color: { argb: `FF${C.text}` } }
  titleCell.alignment = { vertical: "middle", horizontal: "left" }
  styleRange(ws, startRow, startCol, startRow, endCol, (cell) => {
    cell.fill = fillSolid(C.elevated)
    cell.border = thinBorder()
  })

  const headerRow = startRow + 1
  headers.forEach((h, i) => {
    const cell = ws.getCell(headerRow, startCol + i)
    cell.value = h
    cell.font = { name: "微软雅黑", size: 10, bold: true, color: { argb: `FF${C.muted}` } }
    cell.fill = fillSolid(C.page)
    cell.border = thinBorder()
    cell.alignment = { vertical: "middle", horizontal: "left" }
  })

  rows.forEach((row, ri) => {
    row.forEach((val, ci) => {
      const cell = ws.getCell(headerRow + 1 + ri, startCol + ci)
      cell.value = val
      cell.font = { name: "微软雅黑", size: 10, color: { argb: `FF${C.text}` } }
      cell.fill = fillSolid(C.elevated)
      cell.border = thinBorder()
      cell.alignment = { vertical: "middle", horizontal: ci === 0 ? "left" : "left" }
    })
  })

  return headerRow + 1 + rows.length
}

/** 导出与 Web 报表页同构的单 Sheet Excel（含状态图 / 分布图）。 */
export async function exportProjectReport(input: ReportExportInput) {
  const wb = new ExcelJS.Workbook()
  wb.creator = "ningpm"
  wb.created = new Date()
  const ws = wb.addWorksheet("项目报表", {
    views: [{ showGridLines: false }],
    properties: { defaultRowHeight: 18 },
  })

  // A..L
  const widths = [12, 10, 10, 10, 10, 10, 12, 10, 10, 10, 10, 12]
  widths.forEach((w, i) => {
    ws.getColumn(i + 1).width = w
  })

  // page background feel
  for (let r = 1; r <= 60; r++) {
    for (let c = 1; c <= 12; c++) {
      ws.getCell(r, c).fill = fillSolid(C.page)
    }
  }

  const s = input.summary

  // Header
  ws.mergeCells(1, 1, 1, 10)
  const title = ws.getCell(1, 1)
  title.value = input.projectName
  title.font = { name: "微软雅黑", size: 18, bold: true, color: { argb: `FF${C.text}` } }
  title.alignment = { vertical: "middle", horizontal: "left" }
  ws.getRow(1).height = 28

  ws.mergeCells(1, 11, 1, 12)
  const badge = ws.getCell(1, 11)
  badge.value = input.projectKey
  badge.font = { name: "Consolas", size: 11, bold: true, color: { argb: `FF${C.accentInk}` } }
  badge.fill = fillSolid("D6E9FC")
  badge.alignment = { vertical: "middle", horizontal: "center" }
  badge.border = thinBorder()

  ws.mergeCells(2, 1, 2, 12)
  const sub = ws.getCell(2, 1)
  sub.value = `项目报表 · 导出于 ${new Date().toLocaleString("zh-CN")}`
  sub.font = { name: "微软雅黑", size: 10, color: { argb: `FF${C.muted}` } }

  // Summary cards — row 4..6, four cards
  paintCard(ws, 4, 1, 6, 3, "全部事项", String(s.total), `史诗 ${s.roots} · 子事项 ${s.children}`)
  paintCard(ws, 4, 4, 6, 6, "整体完成率", `${s.rate}%`, `已完成 ${s.done} / ${s.total}`)
  paintCard(ws, 4, 7, 6, 9, "子事项完成", `${s.childRate}%`, `${s.childDone} / ${s.children}`)
  paintCard(
    ws,
    4,
    10,
    6,
    12,
    "进行中 / 待验收",
    `${s.doing} / ${s.review}`,
    `待处理 ${s.todo} · 史诗完成 ${s.rootDone}`,
  )
  ws.getRow(4).height = 18
  ws.getRow(5).height = 26
  ws.getRow(6).height = 20

  // Status section title
  ws.mergeCells(8, 1, 8, 12)
  const statusTitle = ws.getCell(8, 1)
  statusTitle.value = "状态分布"
  statusTitle.font = { name: "微软雅黑", size: 12, bold: true, color: { argb: `FF${C.text}` } }
  statusTitle.fill = fillSolid(C.elevated)
  statusTitle.border = thinBorder()
  styleRange(ws, 8, 1, 8, 12, (cell) => {
    cell.fill = fillSolid(C.elevated)
    cell.border = thinBorder()
  })

  const statusChart = await renderStatusChart(input.byStatus, s.total)
  const statusImg = wb.addImage({
    buffer: new Uint8Array(statusChart) as unknown as ExcelJS.Buffer,
    extension: "png",
  })
  // rows 9-16 reserved for chart panel
  styleRange(ws, 9, 1, 16, 12, (cell) => {
    cell.fill = fillSolid(C.elevated)
  })
  for (let c = 1; c <= 12; c++) {
    ws.getCell(9, c).border = {
      top: { style: "thin", color: { argb: `FF${C.border}` } },
      left: c === 1 ? { style: "thin", color: { argb: `FF${C.border}` } } : undefined,
      right: c === 12 ? { style: "thin", color: { argb: `FF${C.border}` } } : undefined,
    }
    ws.getCell(16, c).border = {
      bottom: { style: "thin", color: { argb: `FF${C.border}` } },
      left: c === 1 ? { style: "thin", color: { argb: `FF${C.border}` } } : undefined,
      right: c === 12 ? { style: "thin", color: { argb: `FF${C.border}` } } : undefined,
    }
  }
  for (let r = 10; r <= 15; r++) {
    ws.getCell(r, 1).border = { left: { style: "thin", color: { argb: `FF${C.border}` } } }
    ws.getCell(r, 12).border = { right: { style: "thin", color: { argb: `FF${C.border}` } } }
  }
  ws.addImage(statusImg, {
    tl: { col: 0.15, row: 8.25 },
    ext: { width: 720, height: Math.max(120, 24 + input.byStatus.length * 36) },
    editAs: "oneCell",
  })
  for (let r = 9; r <= 16; r++) ws.getRow(r).height = 18

  // Mini charts for type & cycle
  const typeChart = await renderMiniBars(input.byType, "按类型")
  const cycleChart = await renderMiniBars(input.byCycle, "按迭代")
  const typeImg = wb.addImage({
    buffer: new Uint8Array(typeChart) as unknown as ExcelJS.Buffer,
    extension: "png",
  })
  const cycleImg = wb.addImage({
    buffer: new Uint8Array(cycleChart) as unknown as ExcelJS.Buffer,
    extension: "png",
  })
  styleRange(ws, 18, 1, 26, 6, (cell) => {
    cell.fill = fillSolid(C.elevated)
  })
  styleRange(ws, 18, 7, 26, 12, (cell) => {
    cell.fill = fillSolid(C.elevated)
  })
  for (let r = 18; r <= 26; r++) ws.getRow(r).height = 16
  ws.addImage(typeImg, {
    tl: { col: 0.15, row: 17.2 },
    ext: { width: 340, height: Math.max(80, 36 + input.byType.length * 28) },
    editAs: "oneCell",
  })
  ws.addImage(cycleImg, {
    tl: { col: 6.15, row: 17.2 },
    ext: { width: 340, height: Math.max(80, 36 + input.byCycle.length * 28) },
    editAs: "oneCell",
  })

  // Tables: type + cycle (data, mirrors web tables)
  const typeEnd = writeTable(
    ws,
    28,
    1,
    "按类型",
    ["类型", "总数", "已完成", "完成率"],
    input.byType.map((row) => [row.label, row.total, row.done, rate(row.done, row.total)]),
  )
  writeTable(
    ws,
    28,
    7,
    "按迭代",
    ["迭代", "总数", "已完成", "完成率"],
    input.byCycle.map((row) => [row.label, row.total, row.done, rate(row.done, row.total)]),
  )

  const moduleStart = Math.max(typeEnd, 28 + 1 + input.byCycle.length) + 2
  writeTable(
    ws,
    moduleStart,
    1,
    "按模块",
    ["模块", "总数", "待处理", "进行中", "待验收", "已完成", "完成率"],
    input.byModule.map((row) => [
      row.label,
      row.total,
      row.todo,
      row.doing,
      row.review,
      row.done,
      rate(row.done, row.total),
    ]),
  )

  // Legend
  const legendRow = moduleStart + 2 + Math.max(input.byModule.length, 1) + 1
  ws.mergeCells(legendRow, 1, legendRow, 12)
  const legend = ws.getCell(legendRow, 1)
  legend.value = "图例：绿=已完成 · 橙=待验收 · 蓝=进行中 · 灰=待处理"
  legend.font = { name: "微软雅黑", size: 9, color: { argb: `FF${C.muted}` } }

  const buf = await wb.xlsx.writeBuffer()
  const blob = new Blob([buf], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  const safeKey = (input.projectKey || "project").replace(/[^\w\-]+/g, "_")
  a.download = `${safeKey}-报表-${stamp()}.xlsx`
  a.click()
  URL.revokeObjectURL(url)
}
