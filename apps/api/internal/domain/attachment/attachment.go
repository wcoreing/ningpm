package attachment

import "time"

// IssueAttachment links a resource to an issue.
type IssueAttachment struct {
	ID           string    `json:"id"`
	IssueID      string    `json:"issueId"`
	IssueKey     string    `json:"issueKey,omitempty"`
	IssueTitle   string    `json:"issueTitle,omitempty"`
	ResourceID   string    `json:"resourceId"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	SizeBytes    int64     `json:"sizeBytes"`
	URL          string    `json:"url"`
	CreatedBy    string    `json:"createdBy"`
	CreatorName  string    `json:"creatorName,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ProjectFile is a project-level material (not bound to an issue).
type ProjectFile struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ResourceID   string    `json:"resourceId"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	SizeBytes    int64     `json:"sizeBytes"`
	URL          string    `json:"url"`
	CreatedBy    string    `json:"createdBy"`
	CreatorName  string    `json:"creatorName,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
