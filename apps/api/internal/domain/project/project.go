package project

import "time"

// Project 是事项与成员的顶层容器。
type Project struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatorID   string    `json:"creatorId"`
	IssueCount  int       `json:"issueCount"`
	MyRole      string    `json:"myRole,omitempty"` // 当前用户在该项目中的角色
	CreatedAt   time.Time `json:"createdAt"`
}
