package module

import "time"

// Status is the lifecycle of a product module.
type Status string

const (
	StatusPlanned   Status = "planned"   // 规划中
	StatusActive    Status = "active"    // 进行中
	StatusCompleted Status = "completed" // 已完成
)

// Module groups issues by product scope (Plane Module ≈ 禅道产品模块).
type Module struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      Status    `json:"status"`
	LeadID      string    `json:"leadId,omitempty"`
	LeadName    string    `json:"leadName,omitempty"`
	StartDate   string    `json:"startDate,omitempty"`
	TargetDate  string    `json:"targetDate,omitempty"`
	IssueCount  int       `json:"issueCount"`
	DoneCount   int       `json:"doneCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// StatusLabel returns Chinese status.
func StatusLabel(s Status) string {
	switch s {
	case StatusPlanned:
		return "规划中"
	case StatusActive:
		return "进行中"
	case StatusCompleted:
		return "已完成"
	default:
		return string(s)
	}
}
