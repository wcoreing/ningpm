package issue

import "time"

// Type 统一事项类型（国人习惯：需求 / 任务 / 缺陷）。
type Type string

const (
	TypeStory Type = "story" // 需求
	TypeTask  Type = "task"  // 任务
	TypeBug   Type = "bug"   // 缺陷
	TypeEpic  Type = "epic"  // 史诗
)

// Status 看板执行流状态。
type Status string

const (
	StatusTodo   Status = "todo"   // 待处理
	StatusDoing  Status = "doing"  // 进行中
	StatusReview Status = "review" // 待验收
	StatusDone   Status = "done"   // 已完成
)

// Priority 数字越小越紧急（1 紧急 … 4 低）。
type Priority int

const (
	PriorityUrgent Priority = 1
	PriorityHigh   Priority = 2
	PriorityMedium Priority = 3
	PriorityLow    Priority = 4
)

// Issue 是系统原子工作单元。
type Issue struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"projectId"`
	ProjectKey      string    `json:"projectKey"`
	ProjectName     string    `json:"projectName,omitempty"`
	Number          int       `json:"number"`
	Key             string    `json:"key"` // 如 DEMO-12
	Type            Type      `json:"type"`
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	Status          Status    `json:"status"`
	Priority        Priority  `json:"priority"`
	AssigneeID      string    `json:"assigneeId,omitempty"`
	AssigneeName    string    `json:"assigneeName,omitempty"`
	CycleID         string    `json:"cycleId,omitempty"`
	CycleName       string    `json:"cycleName,omitempty"`
	ModuleID        string    `json:"moduleId,omitempty"`
	ModuleName      string    `json:"moduleName,omitempty"`
	ParentID        string    `json:"parentId,omitempty"`
	ParentKey       string    `json:"parentKey,omitempty"`
	StartDate       string    `json:"startDate,omitempty"` // YYYY-MM-DD 可选
	DueDate         string    `json:"dueDate,omitempty"`   // YYYY-MM-DD 截止日期
	ChildCount      int       `json:"childCount"`
	ChildDone       int       `json:"childDoneCount"`
	AttachmentCount int       `json:"attachmentCount"`
	BoardOrder      float64   `json:"boardOrder"` // 看板排序，越大越靠上；默认取更新时间戳
	CreatorID       string    `json:"creatorId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// TypeLabel 返回中文类型名。
func TypeLabel(t Type) string {
	switch t {
	case TypeStory:
		return "需求"
	case TypeTask:
		return "任务"
	case TypeBug:
		return "缺陷"
	case TypeEpic:
		return "史诗"
	default:
		return string(t)
	}
}

// StatusLabel 返回中文状态名。
func StatusLabel(s Status) string {
	switch s {
	case StatusTodo:
		return "待处理"
	case StatusDoing:
		return "进行中"
	case StatusReview:
		return "待验收"
	case StatusDone:
		return "已完成"
	default:
		return string(s)
	}
}

// IsBoardStatus reports whether the status appears on the execution board.
func IsBoardStatus(s Status) bool {
	switch s {
	case StatusTodo, StatusDoing, StatusReview, StatusDone:
		return true
	default:
		return false
	}
}

// PriorityLabel 返回中文优先级。
func PriorityLabel(p Priority) string {
	switch p {
	case PriorityUrgent:
		return "紧急"
	case PriorityHigh:
		return "高"
	case PriorityMedium:
		return "中"
	case PriorityLow:
		return "低"
	default:
		return "中"
	}
}
