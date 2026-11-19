package store

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ningpm/ningpm/apps/api/internal/domain/issueevent"
)

func (s *Store) ensureIssueEvents() error {
	id := s.idType()
	tm := s.timeCol()
	kind := s.shortCol()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS issue_events (
  id ` + id + ` PRIMARY KEY,
  issue_id ` + id + ` NOT NULL,
  actor_id ` + id + ` NOT NULL DEFAULT '',
  kind ` + kind + ` NOT NULL,
  summary TEXT NOT NULL,
  created_at ` + tm + ` NOT NULL
)`,
		`CREATE INDEX idx_issue_events_issue ON issue_events(issue_id, created_at)`,
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
	return s.backfillIssueEvents()
}

// AddIssueEvent 追加一条操作。id 为空时自动生成；相同 id 重复写入会被忽略。
func (s *Store) AddIssueEvent(ev issueevent.Event) error {
	if strings.TrimSpace(ev.IssueID) == "" || strings.TrimSpace(ev.Summary) == "" {
		return nil
	}
	if ev.ID == "" {
		ev.ID = uuid.NewString()
	}
	if ev.Kind == "" {
		ev.Kind = "update"
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(
		s.insertIgnoreSQL("issue_events(id, issue_id, actor_id, kind, summary, created_at)"),
		ev.ID, ev.IssueID, ev.ActorID, ev.Kind, ev.Summary, ev.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// ListProjectEvents 返回项目内最近的操作，新的在前。
func (s *Store) ListProjectEvents(projectID string, limit int) ([]issueevent.Event, error) {
	if limit <= 0 || limit > 80 {
		limit = 40
	}
	keyExpr := s.sqlConcat(s.projectKeyCol("p"), "'-'", "i.number")
	rows, err := s.db.Query(`
SELECT e.id, e.issue_id, COALESCE(e.actor_id, ''), COALESCE(u.name, ''),
       COALESCE(NULLIF(u.account, ''), COALESCE(u.email, '')), COALESCE(u.avatar, ''),
       e.kind, e.summary, e.created_at, `+keyExpr+`, COALESCE(i.title, '')
FROM issue_events e
JOIN issues i ON i.id = e.issue_id
JOIN projects p ON p.id = i.project_id
LEFT JOIN users u ON u.id = e.actor_id
WHERE i.project_id = ?
ORDER BY e.created_at DESC, e.id DESC
LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []issueevent.Event
	for rows.Next() {
		var ev issueevent.Event
		var created, avatar string
		if err := rows.Scan(
			&ev.ID, &ev.IssueID, &ev.ActorID, &ev.ActorName, &ev.ActorAccount, &avatar,
			&ev.Kind, &ev.Summary, &created, &ev.IssueKey, &ev.IssueTitle,
		); err != nil {
			return nil, err
		}
		if strings.TrimSpace(avatar) != "" && ev.ActorID != "" {
			ev.ActorAvatarURL = "/api/avatars/" + ev.ActorID
		}
		ev.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, ev)
	}
	if out == nil {
		out = []issueevent.Event{}
	}
	return out, rows.Err()
}

// ListIssueEvents 返回事项操作，新的在前。
func (s *Store) ListIssueEvents(issueID string) ([]issueevent.Event, error) {
	rows, err := s.db.Query(`
SELECT e.id, e.issue_id, COALESCE(e.actor_id, ''), COALESCE(u.name, ''),
       COALESCE(NULLIF(u.account, ''), COALESCE(u.email, '')), COALESCE(u.avatar, ''),
       e.kind, e.summary, e.created_at
FROM issue_events e
LEFT JOIN users u ON u.id = e.actor_id
WHERE e.issue_id = ?
ORDER BY e.created_at DESC, e.id DESC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []issueevent.Event
	for rows.Next() {
		var ev issueevent.Event
		var created, avatar string
		if err := rows.Scan(
			&ev.ID, &ev.IssueID, &ev.ActorID, &ev.ActorName, &ev.ActorAccount, &avatar,
			&ev.Kind, &ev.Summary, &created,
		); err != nil {
			return nil, err
		}
		if strings.TrimSpace(avatar) != "" && ev.ActorID != "" {
			ev.ActorAvatarURL = "/api/avatars/" + ev.ActorID
		}
		ev.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, ev)
	}
	if out == nil {
		out = []issueevent.Event{}
	}
	return out, rows.Err()
}

func (s *Store) backfillIssueEvents() error {
	if err := s.backfillCreatedEvents(); err != nil {
		return err
	}
	if err := s.backfillChildEvents(); err != nil {
		return err
	}
	if err := s.backfillCommentEvents(); err != nil {
		return err
	}
	return s.backfillAttachmentEvents()
}

func (s *Store) backfillCreatedEvents() error {
	rows, err := s.db.Query(`SELECT id, COALESCE(creator_id, ''), created_at FROM issues`)
	if err != nil {
		return err
	}
	type row struct{ id, actor, created string }
	var all []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.actor, &item.created); err != nil {
			rows.Close()
			return err
		}
		all = append(all, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range all {
		at, _ := time.Parse(time.RFC3339, item.created)
		if err := s.AddIssueEvent(issueevent.Event{
			ID: "ev-created-" + item.id, IssueID: item.id, ActorID: item.actor, Kind: "created", Summary: "创建了事项", CreatedAt: at,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) backfillChildEvents() error {
	rows, err := s.db.Query(`SELECT id, parent_id, COALESCE(creator_id, ''), title, created_at FROM issues WHERE parent_id IS NOT NULL AND parent_id != ''`)
	if err != nil {
		return err
	}
	type row struct{ id, parent, actor, title, created string }
	var all []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.parent, &item.actor, &item.title, &item.created); err != nil {
			rows.Close()
			return err
		}
		all = append(all, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range all {
		at, _ := time.Parse(time.RFC3339, item.created)
		if err := s.AddIssueEvent(issueevent.Event{
			ID: "ev-child-" + item.id, IssueID: item.parent, ActorID: item.actor, Kind: "child",
			Summary: "添加了子事项「" + clipRunes(item.title, 40) + "」", CreatedAt: at,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) backfillCommentEvents() error {
	rows, err := s.db.Query(`SELECT id, issue_id, author_id, created_at FROM comments`)
	if err != nil {
		return err
	}
	type row struct{ id, issueID, actor, created string }
	var all []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.issueID, &item.actor, &item.created); err != nil {
			rows.Close()
			return err
		}
		all = append(all, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range all {
		at, _ := time.Parse(time.RFC3339, item.created)
		if err := s.AddIssueEvent(issueevent.Event{
			ID: "ev-comment-" + item.id, IssueID: item.issueID, ActorID: item.actor, Kind: "comment", Summary: "发表了评论", CreatedAt: at,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) backfillAttachmentEvents() error {
	rows, err := s.db.Query(`
SELECT a.id, a.issue_id, a.created_by, a.created_at, COALESCE(r.original_name, '')
FROM issue_attachments a
LEFT JOIN resources r ON r.id = a.resource_id`)
	if err != nil {
		return err
	}
	type row struct{ id, issueID, actor, created, name string }
	var all []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.issueID, &item.actor, &item.created, &item.name); err != nil {
			rows.Close()
			return err
		}
		all = append(all, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range all {
		at, _ := time.Parse(time.RFC3339, item.created)
		name := strings.TrimSpace(item.name)
		if name == "" {
			name = "文件"
		}
		if err := s.AddIssueEvent(issueevent.Event{
			ID: "ev-attach-" + item.id, IssueID: item.issueID, ActorID: item.actor, Kind: "attachment",
			Summary: "上传了附件「" + clipRunes(name, 40) + "」", CreatedAt: at,
		}); err != nil {
			return err
		}
	}
	return nil
}

func clipRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
