package issueevent

import "time"

// Event 是事项上的一条操作记录。
type Event struct {
	ID             string    `json:"id"`
	IssueID        string    `json:"issueId"`
	ActorID        string    `json:"actorId,omitempty"`
	ActorName      string    `json:"actorName,omitempty"`
	ActorAccount   string    `json:"actorAccount,omitempty"`
	ActorAvatarURL string    `json:"actorAvatarUrl,omitempty"`
	Kind           string    `json:"kind"`
	Summary        string    `json:"summary"`
	IssueKey       string    `json:"issueKey,omitempty"`
	IssueTitle     string    `json:"issueTitle,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}
