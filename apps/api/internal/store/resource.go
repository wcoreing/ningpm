package store

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ningpm/ningpm/apps/api/internal/domain/attachment"
	"github.com/ningpm/ningpm/apps/api/internal/domain/resource"
	"github.com/ningpm/ningpm/apps/api/internal/domain/user"
)

func (s *Store) ensureResourceTables() error {
	id := s.idType()
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS resources (
  id %s PRIMARY KEY,
  storage_key TEXT NOT NULL,
  original_name TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  uploader_id %s NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(uploader_id) REFERENCES users(id)
)`, id, id),
		`CREATE INDEX idx_resources_uploader ON resources(uploader_id, created_at)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS issue_attachments (
  id %s PRIMARY KEY,
  issue_id %s NOT NULL,
  resource_id %s NOT NULL,
  created_by %s NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(issue_id, resource_id),
  FOREIGN KEY(issue_id) REFERENCES issues(id),
  FOREIGN KEY(resource_id) REFERENCES resources(id),
  FOREIGN KEY(created_by) REFERENCES users(id)
)`, id, id, id, id),
		`CREATE INDEX idx_issue_attachments_issue ON issue_attachments(issue_id, created_at)`,
	}
	for _, stmt := range stmts {
		if strings.HasPrefix(stmt, "CREATE INDEX") {
			if err := s.execIgnoreDup(stmt); err != nil {
				return err
			}
			continue
		}
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// ClearLegacyAvatarFiles wipes non-resource avatar filenames and removes old avatars dir files.
func (s *Store) ClearLegacyAvatarFiles() error {
	rows, err := s.db.Query(`SELECT id, avatar FROM users WHERE avatar != ''`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var clearIDs []string
	for rows.Next() {
		var id, avatar string
		if err := rows.Scan(&id, &avatar); err != nil {
			return err
		}
		// resource ids are UUIDs; old avatar files were uuid.ext
		if strings.Contains(avatar, ".") {
			clearIDs = append(clearIDs, id)
			_ = os.Remove(filepath.Join(filepath.Dir(s.filesDir), "avatars", filepath.Base(avatar)))
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range clearIDs {
		if _, err := s.db.Exec(`UPDATE users SET avatar='' WHERE id=?`, id); err != nil {
			return err
		}
	}
	return nil
}

// FilesDir returns the root directory for resource files.
func (s *Store) FilesDir() string {
	return s.filesDir
}

// ResourceAbsPath resolves an absolute path for a storage key.
func (s *Store) ResourceAbsPath(storageKey string) string {
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" || strings.Contains(storageKey, "..") {
		return ""
	}
	return filepath.Join(s.filesDir, filepath.FromSlash(storageKey))
}

// CreateResourceFromReader stores a new resource on disk and in DB.
func (s *Store) CreateResourceFromReader(
	uploaderID, originalName, contentType string,
	sizeHint int64,
	r io.Reader,
) (*resource.Resource, error) {
	originalName = filepath.Base(strings.TrimSpace(originalName))
	if originalName == "" || originalName == "." {
		originalName = "file"
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	id := uuid.NewString()
	now := time.Now().UTC()
	rel := filepath.ToSlash(filepath.Join(now.Format("2006"), now.Format("01"), id+ext))
	abs := s.ResourceAbsPath(rel)
	if abs == "" {
		return nil, fmt.Errorf("invalid storage key")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(abs)
	if err != nil {
		return nil, err
	}
	n, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(abs)
		return nil, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(abs)
		return nil, closeErr
	}
	if sizeHint > 0 && n == 0 {
		// keep n from copy
	}
	res := &resource.Resource{
		ID:           id,
		StorageKey:   rel,
		OriginalName: originalName,
		ContentType:  contentType,
		SizeBytes:    n,
		UploaderID:   uploaderID,
		CreatedAt:    now,
		URL:          resource.PublicURL(id),
	}
	_, err = s.db.Exec(
		`INSERT INTO resources(id, storage_key, original_name, content_type, size_bytes, uploader_id, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		res.ID, res.StorageKey, res.OriginalName, res.ContentType, res.SizeBytes, res.UploaderID, res.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		_ = os.Remove(abs)
		return nil, err
	}
	return res, nil
}

func (s *Store) GetResource(id string) (*resource.Resource, error) {
	row := s.db.QueryRow(`
SELECT id, storage_key, original_name, content_type, size_bytes, uploader_id, created_at
FROM resources WHERE id=?`, id)
	var r resource.Resource
	var created string
	if err := row.Scan(&r.ID, &r.StorageKey, &r.OriginalName, &r.ContentType, &r.SizeBytes, &r.UploaderID, &created); err != nil {
		return nil, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	r.URL = resource.PublicURL(r.ID)
	return &r, nil
}

func (s *Store) resourceRefCount(resourceID string) (int, error) {
	var n1, n2 int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE avatar=?`, resourceID).Scan(&n1); err != nil {
		return 0, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM issue_attachments WHERE resource_id=?`, resourceID).Scan(&n2); err != nil {
		return 0, err
	}
	return n1 + n2, nil
}

// DeleteResource removes DB row and file when no references remain.
func (s *Store) DeleteResource(id string) error {
	r, err := s.GetResource(id)
	if err != nil {
		return err
	}
	n, err := s.resourceRefCount(id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("resource still referenced")
	}
	if _, err := s.db.Exec(`DELETE FROM resources WHERE id=?`, id); err != nil {
		return err
	}
	if p := s.ResourceAbsPath(r.StorageKey); p != "" {
		_ = os.Remove(p)
	}
	return nil
}

// ResourceAccessFlags describes how a resource is referenced for ACL.
type ResourceAccessFlags struct {
	AsAvatar bool
}

// ResourceAccessInfo returns avatar usage and project ids that attach this resource via issues.
func (s *Store) ResourceAccessInfo(resourceID string) (ResourceAccessFlags, []string, error) {
	var flags ResourceAccessFlags
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE avatar=?`, resourceID).Scan(&n); err != nil {
		return flags, nil, err
	}
	flags.AsAvatar = n > 0

	rows, err := s.db.Query(`
SELECT DISTINCT i.project_id
FROM issue_attachments a
JOIN issues i ON i.id = a.issue_id
WHERE a.resource_id = ?`, resourceID)
	if err != nil {
		return flags, nil, err
	}
	defer rows.Close()
	var projects []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return flags, nil, err
		}
		projects = append(projects, pid)
	}
	return flags, projects, rows.Err()
}

// ForceDeleteResource deletes regardless of refs (caller must clear refs first).
func (s *Store) ForceDeleteResource(id string) error {
	r, err := s.GetResource(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	_, _ = s.db.Exec(`DELETE FROM issue_attachments WHERE resource_id=?`, id)
	_, _ = s.db.Exec(`UPDATE users SET avatar='' WHERE avatar=?`, id)
	if _, err := s.db.Exec(`DELETE FROM resources WHERE id=?`, id); err != nil {
		return err
	}
	if p := s.ResourceAbsPath(r.StorageKey); p != "" {
		_ = os.Remove(p)
	}
	return nil
}

// SetUserAvatarResource sets users.avatar to a resource id and removes the previous one if unused.
func (s *Store) SetUserAvatarResource(userID, resourceID string) (*user.User, error) {
	u, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	old := u.Avatar
	if _, err := s.db.Exec(`UPDATE users SET avatar=? WHERE id=?`, resourceID, userID); err != nil {
		return nil, err
	}
	if old != "" && old != resourceID {
		_ = s.DeleteResource(old) // ignore if still referenced
	}
	return s.GetUserByID(userID)
}

// ClearUserAvatarResource clears avatar pointer and deletes orphan resource.
func (s *Store) ClearUserAvatarResource(userID string) (*user.User, error) {
	u, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	old := u.Avatar
	if _, err := s.db.Exec(`UPDATE users SET avatar='' WHERE id=?`, userID); err != nil {
		return nil, err
	}
	if old != "" {
		_ = s.DeleteResource(old)
	}
	return s.GetUserByID(userID)
}

func (s *Store) ListIssueAttachments(issueID string) ([]attachment.IssueAttachment, error) {
	issueKey := s.sqlConcat(s.projectKeyCol("p"), "'-'", "i.number")
	rows, err := s.db.Query(`
SELECT a.id, a.issue_id, a.resource_id, r.original_name, r.content_type, r.size_bytes,
       a.created_by, COALESCE(u.name,''), a.created_at,
       `+issueKey+`, i.title
FROM issue_attachments a
JOIN resources r ON r.id = a.resource_id
JOIN issues i ON i.id = a.issue_id
JOIN projects p ON p.id = i.project_id
LEFT JOIN users u ON u.id = a.created_by
WHERE a.issue_id = ?
ORDER BY a.created_at ASC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssueAttachments(rows)
}

// ListProjectAttachments returns all attachments under a project, newest first.
func (s *Store) ListProjectAttachments(projectID string) ([]attachment.IssueAttachment, error) {
	issueKey := s.sqlConcat(s.projectKeyCol("p"), "'-'", "i.number")
	rows, err := s.db.Query(`
SELECT a.id, a.issue_id, a.resource_id, r.original_name, r.content_type, r.size_bytes,
       a.created_by, COALESCE(u.name,''), a.created_at,
       `+issueKey+`, i.title
FROM issue_attachments a
JOIN resources r ON r.id = a.resource_id
JOIN issues i ON i.id = a.issue_id
JOIN projects p ON p.id = i.project_id
LEFT JOIN users u ON u.id = a.created_by
WHERE i.project_id = ?
ORDER BY a.created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssueAttachments(rows)
}

func scanIssueAttachments(rows *sql.Rows) ([]attachment.IssueAttachment, error) {
	var out []attachment.IssueAttachment
	for rows.Next() {
		var a attachment.IssueAttachment
		var created string
		if err := rows.Scan(
			&a.ID, &a.IssueID, &a.ResourceID, &a.OriginalName, &a.ContentType, &a.SizeBytes,
			&a.CreatedBy, &a.CreatorName, &created, &a.IssueKey, &a.IssueTitle,
		); err != nil {
			return nil, err
		}
		a.URL = resource.PublicURL(a.ResourceID)
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, a)
	}
	if out == nil {
		out = []attachment.IssueAttachment{}
	}
	return out, rows.Err()
}

func (s *Store) AttachResourceToIssue(issueID, resourceID, createdBy string) (*attachment.IssueAttachment, error) {
	if _, err := s.GetResource(resourceID); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO issue_attachments(id, issue_id, resource_id, created_by, created_at) VALUES(?,?,?,?,?)`,
		id, issueID, resourceID, createdBy, now,
	)
	if err != nil {
		return nil, err
	}
	list, err := s.ListIssueAttachments(issueID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Store) GetIssueAttachment(issueID, attachmentID string) (*attachment.IssueAttachment, error) {
	list, err := s.ListIssueAttachments(issueID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == attachmentID {
			return &list[i], nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Store) DetachIssueAttachment(issueID, attachmentID string) error {
	a, err := s.GetIssueAttachment(issueID, attachmentID)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM issue_attachments WHERE id=? AND issue_id=?`, attachmentID, issueID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	_ = s.DeleteResource(a.ResourceID)
	return nil
}
