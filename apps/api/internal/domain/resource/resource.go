package resource

import "time"

// Resource is a stored file blob with metadata (not bound to a business object).
type Resource struct {
	ID           string    `json:"id"`
	StorageKey   string    `json:"-"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	SizeBytes    int64     `json:"sizeBytes"`
	UploaderID   string    `json:"uploaderId"`
	CreatedAt    time.Time `json:"createdAt"`
	URL          string    `json:"url,omitempty"`
}

// PublicURL returns the authenticated download path.
func PublicURL(id string) string {
	return "/api/resources/" + id
}
