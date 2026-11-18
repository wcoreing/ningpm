export const TYPE_LABELS: Record<string, string> = {
  story: "需求",
  task: "任务",
  bug: "缺陷",
  epic: "史诗",
}

export const STATUS_LABELS: Record<string, string> = {
  todo: "待处理",
  doing: "进行中",
  review: "待验收",
  done: "已完成",
}

export const PRIORITY_LABELS: Record<number, string> = {
  1: "紧急",
  2: "高",
  3: "中",
  4: "低",
}

export const ROLE_LABELS: Record<string, string> = {
  owner: "负责人",
  admin: "管理员",
  member: "成员",
  viewer: "只读",
}

export const CYCLE_STATUS_LABELS: Record<string, string> = {
  planned: "未开始",
  active: "进行中",
  closed: "已完成",
}

export const STATUS_ORDER = ["todo", "doing", "review", "done"] as const

export function typeTone(type: string): "accent" | "neutral" | "danger" | "warning" {
  switch (type) {
    case "story":
      return "accent"
    case "bug":
      return "danger"
    case "task":
      return "neutral"
    case "epic":
      return "warning"
    default:
      return "neutral"
  }
}

export function priorityTone(p: number): "danger" | "warning" | "neutral" | "success" {
  if (p <= 1) return "danger"
  if (p === 2) return "warning"
  if (p === 4) return "success"
  return "neutral"
}
