package comment

import "time"

// Comment is a discussion note on an issue.
type Comment struct {
	ID         string    `json:"id"`
	IssueID    string    `json:"issueId"`
	AuthorID   string    `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
}
