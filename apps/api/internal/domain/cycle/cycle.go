package cycle

import "time"

// Status is the lifecycle of an iteration.
type Status string

const (
	StatusPlanned Status = "planned" // 未开始
	StatusActive  Status = "active"  // 进行中
	StatusClosed  Status = "closed"  // 已完成
)

// Cycle is a project iteration (禅道「迭代」/ Plane Cycle).
type Cycle struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Goal        string    `json:"goal,omitempty"`
	Status      Status    `json:"status"`
	StartDate   string    `json:"startDate"` // YYYY-MM-DD
	EndDate     string    `json:"endDate"`
	IssueCount  int       `json:"issueCount"`
	DoneCount   int       `json:"doneCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// StatusLabel returns Chinese status.
func StatusLabel(s Status) string {
	switch s {
	case StatusPlanned:
		return "未开始"
	case StatusActive:
		return "进行中"
	case StatusClosed:
		return "已完成"
	default:
		return string(s)
	}
}
