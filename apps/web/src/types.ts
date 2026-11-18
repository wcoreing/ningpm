export type Project = {
  id: string
  key: string
  name: string
  description?: string
  creatorId: string
  issueCount: number
  myRole?: "owner" | "admin" | "member" | "viewer"
  createdAt: string
}

export type Issue = {
  id: string
  projectId: string
  projectKey: string
  projectName?: string
  number: number
  key: string
  type: "story" | "task" | "bug" | "epic"
  title: string
  description?: string
  status: "todo" | "doing" | "review" | "done"
  priority: number
  assigneeId?: string
  assigneeName?: string
  cycleId?: string
  cycleName?: string
  moduleId?: string
  moduleName?: string
  parentId?: string
  parentKey?: string
  childCount?: number
  childDoneCount?: number
  attachmentCount?: number
  creatorId: string
  createdAt: string
  updatedAt: string
}

export type Module = {
  id: string
  projectId: string
  name: string
  description?: string
  status: "planned" | "active" | "completed"
  leadId?: string
  leadName?: string
  startDate?: string
  targetDate?: string
  issueCount: number
  doneCount: number
  createdAt: string
}

export type Cycle = {
  id: string
  projectId: string
  name: string
  goal?: string
  status: "planned" | "active" | "closed"
  startDate: string
  endDate: string
  issueCount: number
  doneCount: number
  createdAt: string
}

export type UserOption = {
  id: string
  name: string
  email: string
  avatarUrl?: string
}

export type IssueComment = {
  id: string
  issueId: string
  authorId: string
  authorName: string
  body: string
  createdAt: string
}

export type ProjectMember = {
  projectId: string
  userId: string
  name: string
  email: string
  avatarUrl?: string
  role: "owner" | "admin" | "member" | "viewer"
  joinedAt: string
}

export type NotificationKind = "assigned" | "commented"

export type AppNotification = {
  id: string
  userId: string
  actorId: string
  actorName: string
  kind: NotificationKind
  issueId: string
  issueKey: string
  issueTitle: string
  projectId: string
  projectName: string
  body?: string
  readAt?: string | null
  createdAt: string
}
