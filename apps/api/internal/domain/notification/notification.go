package notification

import "time"

// Kind classifies why the user was notified.
type Kind string

const (
	KindAssigned  Kind = "assigned"
	KindCommented Kind = "commented"
)

// Notification is an in-app alert for a user.
type Notification struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	ActorID     string     `json:"actorId"`
	ActorName   string     `json:"actorName"`
	Kind        Kind       `json:"kind"`
	IssueID     string     `json:"issueId"`
	IssueKey    string     `json:"issueKey"`
	IssueTitle  string     `json:"issueTitle"`
	ProjectID   string     `json:"projectId"`
	ProjectName string     `json:"projectName"`
	Body        string     `json:"body,omitempty"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}
