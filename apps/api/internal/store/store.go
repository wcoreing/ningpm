package store

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"github.com/ningpm/ningpm/apps/api/internal/domain/comment"
	"github.com/ningpm/ningpm/apps/api/internal/domain/cycle"
	"github.com/ningpm/ningpm/apps/api/internal/domain/issue"
	"github.com/ningpm/ningpm/apps/api/internal/domain/member"
	"github.com/ningpm/ningpm/apps/api/internal/domain/module"
	"github.com/ningpm/ningpm/apps/api/internal/domain/notification"
	"github.com/ningpm/ningpm/apps/api/internal/domain/project"
	"github.com/ningpm/ningpm/apps/api/internal/domain/user"
)

// Options configures persistence.
type Options struct {
	Driver     string // sqlite | mysql
	SQLitePath string
	MySQLDSN   string
	FilesDir   string
}

// Store is the SQL persistence layer (sqlite locally, mysql in production).
type Store struct {
	db       *sql.DB
	driver   string
	filesDir string
}

// Open opens the database and migrates schema.
func Open(opt Options) (*Store, error) {
	driver := strings.ToLower(strings.TrimSpace(opt.Driver))
	if driver == "" {
		driver = "sqlite"
	}
	filesDir := strings.TrimSpace(opt.FilesDir)
	if filesDir == "" {
		if driver == "mysql" {
			filesDir = filepath.Join("data", "files")
		} else {
			filesDir = filepath.Join(filepath.Dir(opt.SQLitePath), "files")
		}
	}
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return nil, err
	}

	var (
		db  *sql.DB
		err error
	)
	switch driver {
	case "mysql":
		if strings.TrimSpace(opt.MySQLDSN) == "" {
			return nil, fmt.Errorf("NINGPM_DB_DSN required for mysql")
		}
		dsn := opt.MySQLDSN
		if !strings.Contains(dsn, "multiStatements=") {
			if strings.Contains(dsn, "?") {
				dsn += "&multiStatements=true"
			} else {
				dsn += "?multiStatements=true"
			}
		}
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(time.Hour)
	case "sqlite":
		dir := filepath.Dir(opt.SQLitePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		db, err = sql.Open("sqlite", opt.SQLitePath)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(1)
	default:
		return nil, fmt.Errorf("unsupported db driver: %s", driver)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db, driver: driver, filesDir: filesDir}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// idType is VARCHAR on MySQL (TEXT cannot be PRIMARY KEY without length).
func (s *Store) idType() string {
	if s.driver == "mysql" {
		return "VARCHAR(64)"
	}
	return "TEXT"
}

// textDef is a text-ish column that may carry DEFAULT '' (MySQL forbids DEFAULT on TEXT).
func (s *Store) textDef() string {
	if s.driver == "mysql" {
		return "VARCHAR(4096) NOT NULL DEFAULT ''"
	}
	return "TEXT NOT NULL DEFAULT ''"
}

// timeCol stores RFC3339 timestamps; VARCHAR so MySQL can index them.
func (s *Store) timeCol() string {
	if s.driver == "mysql" {
		return "VARCHAR(64)"
	}
	return "TEXT"
}

// shortCol is for status/type/role/kind values that may appear in indexes.
func (s *Store) shortCol() string {
	if s.driver == "mysql" {
		return "VARCHAR(32)"
	}
	return "TEXT"
}

// projectKeyCol returns a qualified projects.key reference that is safe on MySQL.
func (s *Store) projectKeyCol(alias string) string {
	if alias == "" {
		if s.driver == "mysql" {
			return "`key`"
		}
		return "key"
	}
	if s.driver == "mysql" {
		return alias + ".`key`"
	}
	return alias + ".key"
}

// sqlConcat joins SQL expressions with || (sqlite) or CONCAT (mysql).
func (s *Store) sqlConcat(parts ...string) string {
	if len(parts) == 0 {
		return "''"
	}
	if s.driver == "mysql" {
		return "CONCAT(" + strings.Join(parts, ", ") + ")"
	}
	return strings.Join(parts, " || ")
}

// orderByNameAsc returns a case-insensitive name sort expression for the current driver.
func (s *Store) orderByNameAsc(expr string) string {
	if s.driver == "mysql" {
		return "ORDER BY " + expr + " ASC"
	}
	return "ORDER BY " + expr + " COLLATE NOCASE ASC"
}

// insertIgnoreSQL builds INSERT IGNORE / INSERT OR IGNORE for column list like `t(a,b)`.
func (s *Store) insertIgnoreSQL(tableCols string) string {
	placeholders := "?"
	// count commas in "a,b,c" portion
	n := 1
	if i := strings.IndexByte(tableCols, '('); i >= 0 {
		inner := tableCols[i+1:]
		if j := strings.IndexByte(inner, ')'); j >= 0 {
			inner = inner[:j]
		}
		if strings.TrimSpace(inner) != "" {
			n = strings.Count(inner, ",") + 1
		}
	}
	for i := 1; i < n; i++ {
		placeholders += ",?"
	}
	if s.driver == "mysql" {
		return "INSERT IGNORE INTO " + tableCols + " VALUES(" + placeholders + ")"
	}
	return "INSERT OR IGNORE INTO " + tableCols + " VALUES(" + placeholders + ")"
}

func (s *Store) execIgnoreDup(sqlStmt string) error {
	_, err := s.db.Exec(sqlStmt)
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	// MySQL 1061 duplicate key / index; sqlite already has IF NOT EXISTS
	if strings.Contains(msg, "duplicate") || strings.Contains(msg, "already exists") {
		return nil
	}
	return err
}

func (s *Store) migrate() error {
	id := s.idType()
	td := s.textDef()
	tm := s.timeCol()
	sh := s.shortCol()
	keyCol := "key"
	if s.driver == "mysql" {
		keyCol = "`key`"
	}
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
  id %s PRIMARY KEY,
  name TEXT NOT NULL,
  account %s NOT NULL UNIQUE,
  email %s,
  password_hash TEXT NOT NULL,
  created_at %s NOT NULL
)`, id, id, id, tm),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS projects (
  id %s PRIMARY KEY,
  %s %s NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description %s,
  creator_id %s NOT NULL,
  created_at %s NOT NULL
)`, id, keyCol, id, td, id, tm),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS issues (
  id %s PRIMARY KEY,
  project_id %s NOT NULL,
  number INTEGER NOT NULL,
  type %s NOT NULL,
  title TEXT NOT NULL,
  description %s,
  status %s NOT NULL,
  priority INTEGER NOT NULL DEFAULT 3,
  assignee_id %s NULL,
  creator_id %s NOT NULL,
  created_at %s NOT NULL,
  updated_at %s NOT NULL,
  UNIQUE(project_id, number),
  FOREIGN KEY(project_id) REFERENCES projects(id)
)`, id, id, sh, td, sh, id, id, tm, tm),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS comments (
  id %s PRIMARY KEY,
  issue_id %s NOT NULL,
  author_id %s NOT NULL,
  body TEXT NOT NULL,
  created_at %s NOT NULL,
  FOREIGN KEY(issue_id) REFERENCES issues(id)
)`, id, id, id, tm),
		`CREATE INDEX idx_comments_issue ON comments(issue_id, created_at)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS project_members (
  project_id %s NOT NULL,
  user_id %s NOT NULL,
  role %s NOT NULL,
  joined_at %s NOT NULL,
  PRIMARY KEY(project_id, user_id),
  FOREIGN KEY(project_id) REFERENCES projects(id),
  FOREIGN KEY(user_id) REFERENCES users(id)
)`, id, id, sh, tm),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS cycles (
  id %s PRIMARY KEY,
  project_id %s NOT NULL,
  name TEXT NOT NULL,
  goal %s,
  status %s NOT NULL,
  start_date %s NOT NULL,
  end_date %s NOT NULL,
  created_at %s NOT NULL,
  FOREIGN KEY(project_id) REFERENCES projects(id)
)`, id, id, td, sh, tm, tm, tm),
		`CREATE INDEX idx_cycles_project ON cycles(project_id, status)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS modules (
  id %s PRIMARY KEY,
  project_id %s NOT NULL,
  name TEXT NOT NULL,
  description %s,
  status %s NOT NULL,
  lead_id %s NULL,
  start_date %s,
  target_date %s,
  created_at %s NOT NULL,
  FOREIGN KEY(project_id) REFERENCES projects(id)
)`, id, id, td, sh, id, td, td, tm),
		`CREATE INDEX idx_modules_project ON modules(project_id, status)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS notifications (
  id %s PRIMARY KEY,
  user_id %s NOT NULL,
  actor_id %s NOT NULL,
  kind %s NOT NULL,
  issue_id %s NOT NULL,
  project_id %s NOT NULL,
  body %s,
  read_at %s NULL,
  created_at %s NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(actor_id) REFERENCES users(id),
  FOREIGN KEY(issue_id) REFERENCES issues(id),
  FOREIGN KEY(project_id) REFERENCES projects(id)
)`, id, id, id, sh, id, id, td, tm, tm),
		`CREATE INDEX idx_notifications_user ON notifications(user_id, created_at)`,
		`CREATE INDEX idx_notifications_unread ON notifications(user_id, read_at)`,
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
	if err := s.ensureIssueCycleColumn(); err != nil {
		return err
	}
	if err := s.ensureIssueModuleColumn(); err != nil {
		return err
	}
	if err := s.ensureIssueParentColumn(); err != nil {
		return err
	}
	if err := s.ensureIssueScheduleColumns(); err != nil {
		return err
	}
	if err := s.ensureIssueBoardOrderColumn(); err != nil {
		return err
	}
	if err := s.ensureUserAvatarColumn(); err != nil {
		return err
	}
	if err := s.ensureUserAccountColumn(); err != nil {
		return err
	}
	if err := s.ensureUserIsAdminColumn(); err != nil {
		return err
	}
	if err := s.ensureResourceTables(); err != nil {
		return err
	}
	if err := s.ClearLegacyAvatarFiles(); err != nil {
		return err
	}
	if err := s.backfillProjectOwners(); err != nil {
		return err
	}
	if err := s.ensureDemoMembers(); err != nil {
		return err
	}
	if err := s.ensureDemoInvitePool(); err != nil {
		return err
	}
	if err := s.ensureDemoCycle(); err != nil {
		return err
	}
	if err := s.clearLegacyInboxStatuses(); err != nil {
		return err
	}
	if err := s.ensureDemoModules(); err != nil {
		return err
	}
	if err := s.ensureDemoNotifications(); err != nil {
		return err
	}
	return s.ensureDemoSubIssues()
}

// clearLegacyInboxStatuses migrates retired triage/rejected issues onto the board.
func (s *Store) clearLegacyInboxStatuses() error {
	_, err := s.db.Exec(
		`UPDATE issues SET status = ?, updated_at = ? WHERE status IN ('triage', 'rejected')`,
		issue.StatusTodo, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) ensureIssueCycleColumn() error {
	return s.ensureIssueColumn("cycle_id")
}

func (s *Store) ensureIssueModuleColumn() error {
	return s.ensureIssueColumn("module_id")
}

func (s *Store) ensureIssueParentColumn() error {
	return s.ensureIssueColumn("parent_id")
}

func (s *Store) ensureIssueScheduleColumns() error {
	if err := s.ensureIssueTextColumn("due_date"); err != nil {
		return err
	}
	return s.ensureIssueTextColumn("start_date")
}

// ensureIssueBoardOrderColumn adds board_order and backfills from updated_at (last-modified default).
func (s *Store) ensureIssueBoardOrderColumn() error {
	has, err := s.hasColumn("issues", "board_order")
	if err != nil {
		return err
	}
	if !has {
		if _, err := s.db.Exec(`ALTER TABLE issues ADD COLUMN board_order REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		rows, err := s.db.Query(`SELECT id, updated_at FROM issues`)
		if err != nil {
			return err
		}
		defer rows.Close()
		type row struct {
			id      string
			updated string
		}
		var list []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.updated); err != nil {
				return err
			}
			list = append(list, r)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, r := range list {
			t, err := time.Parse(time.RFC3339, r.updated)
			if err != nil {
				t = time.Now().UTC()
			}
			if _, err := s.db.Exec(`UPDATE issues SET board_order=? WHERE id=?`, float64(t.UnixMilli()), r.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) ensureIssueTextColumn(col string) error {
	has, err := s.hasColumn("issues", col)
	if err != nil || has {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE issues ADD COLUMN ` + col + ` TEXT NULL`)
	return err
}

func (s *Store) hasColumn(table, col string) (bool, error) {
	if s.driver == "mysql" {
		var n int
		err := s.db.QueryRow(
			`SELECT COUNT(1) FROM information_schema.COLUMNS
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
			table, col,
		).Scan(&n)
		return n > 0, err
	}
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == col {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Store) ensureUserAvatarColumn() error {
	has, err := s.hasColumn("users", "avatar")
	if err != nil || has {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE users ADD COLUMN avatar ` + s.idType() + ` NOT NULL DEFAULT ''`)
	return err
}

func (s *Store) ensureUserIsAdminColumn() error {
	has, err := s.hasColumn("users", "is_admin")
	if err != nil {
		return err
	}
	if !has {
		col := "INTEGER NOT NULL DEFAULT 0"
		if s.driver == "mysql" {
			col = "TINYINT(1) NOT NULL DEFAULT 0"
		}
		if _, err := s.db.Exec(`ALTER TABLE users ADD COLUMN is_admin ` + col); err != nil {
			return err
		}
	}
	// 引导账号：历史 admin@ningpm.local 自动升级为系统管理员
	_, err = s.db.Exec(`UPDATE users SET is_admin = 1 WHERE email = ? OR account = ?`, "admin@ningpm.local", "admin@ningpm.local")
	return err
}

// ensureUserAccountColumn adds login account and backfills from legacy email.
func (s *Store) ensureUserAccountColumn() error {
	has, err := s.hasColumn("users", "account")
	if err != nil {
		return err
	}
	if !has {
		// MySQL forbids DEFAULT on TEXT; use VARCHAR for login ids (also index-friendly).
		col := "TEXT NOT NULL DEFAULT ''"
		if s.driver == "mysql" {
			col = "VARCHAR(191) NOT NULL DEFAULT ''"
		}
		if _, err := s.db.Exec(`ALTER TABLE users ADD COLUMN account ` + col); err != nil {
			return err
		}
	}
	// backfill from email for rows still empty
	if _, err := s.db.Exec(`UPDATE users SET account = email WHERE account = '' OR account IS NULL`); err != nil {
		return err
	}
	// unique index (ignore if exists)
	_, _ = s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_account ON users(account)`)
	if s.driver == "mysql" {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM information_schema.STATISTICS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND INDEX_NAME = 'idx_users_account'`).Scan(&n)
		if n == 0 {
			_, _ = s.db.Exec(`CREATE UNIQUE INDEX idx_users_account ON users(account)`)
		}
	}
	return nil
}

func (s *Store) ensureIssueColumn(col string) error {
	has, err := s.hasColumn("issues", col)
	if err != nil || has {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE issues ADD COLUMN ` + col + ` ` + s.idType() + ` NULL`)
	return err
}

// ensureDemoModules seeds Plane-style / 禅道-style product modules.
func (s *Store) ensureDemoModules() error {
	var projectID, adminID string
	err := s.db.QueryRow(`SELECT id, creator_id FROM projects WHERE `+s.projectKeyCol("")+` = 'DEMO'`).Scan(&projectID, &adminID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM modules WHERE project_id = ?`, projectID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	mods := []struct {
		name, desc string
		status     module.Status
	}{
		{"账号与权限", "登录、成员、角色", module.StatusActive},
		{"看板协作", "列表、看板、拖拽与详情", module.StatusActive},
		{"报表导出", "周报与数据导出（规划中）", module.StatusPlanned},
	}
	var boardModuleID string
	for i, m := range mods {
		id := uuid.NewString()
		if i == 1 {
			boardModuleID = id
		}
		if _, err := s.db.Exec(
			`INSERT INTO modules(id, project_id, name, description, status, lead_id, start_date, target_date, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			id, projectID, m.name, m.desc, m.status, adminID, "", "", now,
		); err != nil {
			return err
		}
	}
	if boardModuleID != "" {
		_, _ = s.db.Exec(
			`UPDATE issues SET module_id = ? WHERE project_id = ? AND status != ? AND (module_id IS NULL OR module_id = '')`,
			boardModuleID, projectID, issue.StatusDone,
		)
	}
	return nil
}

// backfillProjectOwners ensures legacy projects have at least a creator owner.
func (s *Store) backfillProjectOwners() error {
	rows, err := s.db.Query(`
SELECT p.id, p.creator_id FROM projects p
WHERE NOT EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = p.id)`)
	if err != nil {
		return err
	}
	type pair struct{ projectID, creatorID string }
	var pending []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.projectID, &p.creatorID); err != nil {
			_ = rows.Close()
			return err
		}
		pending = append(pending, p)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, p := range pending {
		if _, err := s.db.Exec(
			s.insertIgnoreSQL(`project_members(project_id, user_id, role, joined_at)`),
			p.projectID, p.creatorID, member.RoleOwner, now,
		); err != nil {
			return err
		}
	}
	return nil
}

// ensureDemoMembers adds the demo PM account into DEMO when both exist.
func (s *Store) ensureDemoMembers() error {
	var projectID string
	err := s.db.QueryRow(`SELECT id FROM projects WHERE ` + s.projectKeyCol("") + ` = 'DEMO'`).Scan(&projectID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var pmID string
	err = s.db.QueryRow(`SELECT id FROM users WHERE email = 'pm@ningpm.local'`).Scan(&pmID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		s.insertIgnoreSQL(`project_members(project_id, user_id, role, joined_at)`),
		projectID, pmID, member.RoleMember, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// ensureDemoInvitePool creates system users who are NOT DEMO members, so invite autocomplete has hits.
func (s *Store) ensureDemoInvitePool() error {
	type seedUser struct {
		name, email, password string
	}
	seeds := []seedUser{
		{"设计师小陈", "design@ningpm.local", "design123"},
		{"测试小李", "qa@ningpm.local", "qa123456"},
		{"运营小周", "ops@ningpm.local", "ops123456"},
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, su := range seeds {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE email = ?`, su.email).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(su.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO users(id, name, account, email, password_hash, is_admin, created_at) VALUES(?,?,?,?,?,?,?)`,
			uuid.NewString(), su.name, su.email, su.email, string(hash), 0, now,
		); err != nil {
			return err
		}
	}
	return nil
}

// ensureDemoCycle creates an active “本周迭代” for DEMO if missing.
func (s *Store) ensureDemoCycle() error {
	var projectID string
	err := s.db.QueryRow(`SELECT id FROM projects WHERE ` + s.projectKeyCol("") + ` = 'DEMO'`).Scan(&projectID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM cycles WHERE project_id = ?`, projectID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -int(now.Weekday())+1) // Monday-ish
	if now.Weekday() == time.Sunday {
		start = now.AddDate(0, 0, -6)
	}
	end := start.AddDate(0, 0, 6)
	cycleID := uuid.NewString()
	if _, err := s.db.Exec(
		`INSERT INTO cycles(id, project_id, name, goal, status, start_date, end_date, created_at) VALUES(?,?,?,?,?,?,?,?)`,
		cycleID, projectID, "本周迭代", "把成员与看板主路径跑通", cycle.StatusActive,
		start.Format("2006-01-02"), end.Format("2006-01-02"), now.Format(time.RFC3339),
	); err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE issues SET cycle_id = ? WHERE project_id = ? AND status != ? AND cycle_id IS NULL`,
		cycleID, projectID, issue.StatusDone,
	)
	return err
}

// SeedDemo creates default admin + sample project/issues when admin is missing.
// Open() may have already inserted invite-pool users, so we key off admin email—not empty table.
func (s *Store) SeedDemo() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE email = ?`, "admin@ningpm.local").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	adminID := uuid.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`INSERT INTO users(id, name, account, email, password_hash, is_admin, created_at) VALUES(?,?,?,?,?,?,?)`,
		adminID, "管理员", "admin@ningpm.local", "admin@ningpm.local", string(hash), 1, now.Format(time.RFC3339),
	); err != nil {
		return err
	}

	pmID := uuid.NewString()
	pmHash, _ := bcrypt.GenerateFromPassword([]byte("pm123456"), bcrypt.DefaultCost)
	if _, err := s.db.Exec(
		`INSERT INTO users(id, name, account, email, password_hash, is_admin, created_at) VALUES(?,?,?,?,?,?,?)`,
		pmID, "产品经理", "pm@ningpm.local", "pm@ningpm.local", string(pmHash), 0, now.Format(time.RFC3339),
	); err != nil {
		return err
	}

	projID := uuid.NewString()
	if _, err := s.db.Exec(
		`INSERT INTO projects(id, `+s.projectKeyCol("")+`, name, description, creator_id, created_at) VALUES(?,?,?,?,?,?)`,
		projID, "DEMO", "演示项目", "借鉴禅道术语与 Plane 交互的示例项目", adminID, now.Format(time.RFC3339),
	); err != nil {
		return err
	}
	if _, err := s.db.Exec(
		`INSERT INTO project_members(project_id, user_id, role, joined_at) VALUES(?,?,?,?)`,
		projID, adminID, member.RoleOwner, now.Format(time.RFC3339),
	); err != nil {
		return err
	}
	if _, err := s.db.Exec(
		`INSERT INTO project_members(project_id, user_id, role, joined_at) VALUES(?,?,?,?)`,
		projID, pmID, member.RoleMember, now.Format(time.RFC3339),
	); err != nil {
		return err
	}

	samples := []struct {
		typ      issue.Type
		title    string
		status   issue.Status
		priority issue.Priority
		assignee string
	}{
		{issue.TypeStory, "支持项目创建与成员邀请", issue.StatusDoing, issue.PriorityHigh, adminID},
		{issue.TypeTask, "完善事项列表筛选", issue.StatusTodo, issue.PriorityMedium, pmID},
		{issue.TypeBug, "看板拖拽后状态未持久化", issue.StatusReview, issue.PriorityUrgent, adminID},
		{issue.TypeStory, "我的待办视图", issue.StatusTodo, issue.PriorityMedium, adminID},
		{issue.TypeTask, "默认中文文案与状态名", issue.StatusDone, issue.PriorityLow, pmID},
	}
	start := now.AddDate(0, 0, -int(now.Weekday())+1)
	if now.Weekday() == time.Sunday {
		start = now.AddDate(0, 0, -6)
	}
	end := start.AddDate(0, 0, 6)
	cycleID := uuid.NewString()
	if _, err := s.db.Exec(
		`INSERT INTO cycles(id, project_id, name, goal, status, start_date, end_date, created_at) VALUES(?,?,?,?,?,?,?,?)`,
		cycleID, projID, "本周迭代", "把成员与看板主路径跑通", cycle.StatusActive,
		start.Format("2006-01-02"), end.Format("2006-01-02"), now.Format(time.RFC3339),
	); err != nil {
		return err
	}

	var firstIssueID string
	for i, sample := range samples {
		id := uuid.NewString()
		if i == 0 {
			firstIssueID = id
		}
		ts := now.Add(time.Duration(i) * time.Minute).Format(time.RFC3339)
		desc := ""
		if sample.typ == issue.TypeBug {
			desc = "复现步骤：\n1. 打开看板\n2. 拖拽卡片到「进行中」\n3. 刷新后状态回退"
		}
		var cid any
		if sample.status != issue.StatusDone {
			cid = cycleID
		}
		if _, err := s.db.Exec(
			`INSERT INTO issues(id, project_id, number, type, title, description, status, priority, assignee_id, creator_id, created_at, updated_at, cycle_id)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, projID, i+1, sample.typ, sample.title, desc, sample.status, sample.priority, sample.assignee, adminID, ts, ts, cid,
		); err != nil {
			return err
		}
	}
	if firstIssueID != "" {
		_, _ = s.db.Exec(
			`INSERT INTO comments(id, issue_id, author_id, body, created_at) VALUES(?,?,?,?,?)`,
			uuid.NewString(), firstIssueID, pmID, "先把创建流程做通，成员邀请可以下一迭代补。", now.Add(10*time.Minute).Format(time.RFC3339),
		)
	}
	return nil
}

// --- users ---

const userSelectCols = `id, name, COALESCE(account,''), COALESCE(email,''), COALESCE(avatar,''), password_hash, COALESCE(is_admin, 0), created_at`

func (s *Store) GetUserByAccount(account string) (*user.User, error) {
	account = strings.TrimSpace(account)
	row := s.db.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE account = ? OR ((account = '' OR account IS NULL) AND email = ?)`, account, account)
	return scanUser(row)
}

// GetUserByEmail is a legacy alias for GetUserByAccount.
func (s *Store) GetUserByEmail(email string) (*user.User, error) {
	return s.GetUserByAccount(email)
}

func (s *Store) GetUserByID(id string) (*user.User, error) {
	row := s.db.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// CreateUser inserts a normal user.
func (s *Store) CreateUser(name, account, plainPassword string) (*user.User, error) {
	return s.CreateUserAdmin(name, account, plainPassword, false)
}

// AllocateLoginAccount returns a unique login id. Prefer account if non-empty;
// otherwise derive from display name.
func (s *Store) AllocateLoginAccount(account, name string) (string, error) {
	account = strings.TrimSpace(account)
	if account != "" {
		if _, err := s.GetUserByAccount(account); err == nil {
			return "", fmt.Errorf("account taken")
		} else if err != sql.ErrNoRows {
			return "", err
		}
		return account, nil
	}
	base := strings.TrimSpace(name)
	if base == "" {
		base = "user"
	}
	base = strings.Join(strings.Fields(base), "")
	candidate := base
	for i := 0; i < 20; i++ {
		if _, err := s.GetUserByAccount(candidate); err == sql.ErrNoRows {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
		candidate = fmt.Sprintf("%s_%s", base, uuid.NewString()[:4])
	}
	return fmt.Sprintf("%s_%s", base, uuid.NewString()[:8]), nil
}

// CreateUserAdmin creates a user and optionally marks them as system admin.
func (s *Store) CreateUserAdmin(name, account, plainPassword string, isAdmin bool) (*user.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	login := strings.TrimSpace(account)
	u := &user.User{
		ID:           uuid.NewString(),
		Name:         strings.TrimSpace(name),
		Account:      login,
		Email:        "",
		PasswordHash: string(hash),
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now().UTC(),
	}
	if u.Name == "" {
		u.Name = login
	}
	adminVal := 0
	if isAdmin {
		adminVal = 1
	}
	// legacy schemas keep email NOT NULL UNIQUE; mirror account when contact email omitted
	email := u.Email
	if email == "" {
		email = u.Account
		u.Email = email
	}
	_, err = s.db.Exec(
		`INSERT INTO users(id, name, account, email, password_hash, is_admin, created_at) VALUES(?,?,?,?,?,?,?)`,
		u.ID, u.Name, u.Account, email, u.PasswordHash, adminVal, u.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUserName updates a user's display name.
func (s *Store) UpdateUserName(userID, name string) (*user.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name required")
	}
	// 先确认用户存在；勿用 RowsAffected==0 判不存在（MySQL 同值 UPDATE 常返回 0）
	if _, err := s.GetUserByID(userID); err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`UPDATE users SET name=? WHERE id=?`, name, userID); err != nil {
		return nil, err
	}
	return s.GetUserByID(userID)
}

// UpdateUserPassword sets a new password hash after verifying the current one.
func (s *Store) UpdateUserPassword(userID, currentPassword, newPassword string) error {
	u, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)) != nil {
		return fmt.Errorf("current password mismatch")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, string(hash), userID)
	return err
}

// EnsureUserByAccount returns an existing user, or creates one with a temporary password.
func (s *Store) EnsureUserByAccount(account, name string) (u *user.User, tempPassword string, created bool, err error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, "", false, fmt.Errorf("account required")
	}
	u, err = s.GetUserByAccount(account)
	if err == nil {
		return u, "", false, nil
	}
	if err != sql.ErrNoRows {
		return nil, "", false, err
	}
	tempPassword = randomPassword(10)
	if strings.TrimSpace(name) == "" {
		name = account
	}
	u, err = s.CreateUser(name, account, tempPassword)
	if err != nil {
		return nil, "", false, err
	}
	return u, tempPassword, true, nil
}

// EnsureUserByEmail is a legacy alias for EnsureUserByAccount.
func (s *Store) EnsureUserByEmail(email, name string) (u *user.User, tempPassword string, created bool, err error) {
	return s.EnsureUserByAccount(email, name)
}

func randomPassword(n int) string {
	const alphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return uuid.NewString()[:n]
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = alphabet[int(raw[i])%len(alphabet)]
	}
	return string(out)
}

func (s *Store) ListUsers() ([]user.User, error) {
	rows, err := s.db.Query(`SELECT ` + userSelectCols + ` FROM users ORDER BY is_admin DESC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []user.User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// UpdateUserAdmin updates profile fields for system user management.
func (s *Store) UpdateUserAdmin(userID, name, account, newPassword string, isAdmin *bool) (*user.User, error) {
	u, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = u.Name
	}
	account = strings.TrimSpace(account)
	if account == "" {
		account = u.Account
		if account == "" {
			account = u.Email
		}
	}
	cur := u.Account
	if cur == "" {
		cur = u.Email
	}
	if account != cur {
		if other, err := s.GetUserByAccount(account); err == nil && other.ID != userID {
			return nil, fmt.Errorf("account taken")
		} else if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}
	adminVal := 0
	if u.IsAdmin {
		adminVal = 1
	}
	if isAdmin != nil {
		if *isAdmin {
			adminVal = 1
		} else {
			adminVal = 0
		}
	}
	if _, err := s.db.Exec(
		`UPDATE users SET name=?, account=?, is_admin=? WHERE id=?`,
		name, account, adminVal, userID,
	); err != nil {
		return nil, err
	}
	if pwd := strings.TrimSpace(newPassword); pwd != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		if _, err := s.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, string(hash), userID); err != nil {
			return nil, err
		}
	}
	return s.GetUserByID(userID)
}

// CountAdmins returns how many system admins exist.
func (s *Store) CountAdmins() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE is_admin = 1`).Scan(&n)
	return n, err
}

// DeleteUser removes a user; non-nullable author/creator refs are moved to fallbackUserID.
func (s *Store) DeleteUser(userID, fallbackUserID string) error {
	if strings.TrimSpace(fallbackUserID) == "" || fallbackUserID == userID {
		return fmt.Errorf("invalid fallback user")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM project_members WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM notifications WHERE user_id = ? OR actor_id = ?`, userID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE issues SET assignee_id = NULL WHERE assignee_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE modules SET lead_id = NULL WHERE lead_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE projects SET creator_id = ? WHERE creator_id = ?`, fallbackUserID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE issues SET creator_id = ? WHERE creator_id = ?`, fallbackUserID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE comments SET author_id = ? WHERE author_id = ?`, fallbackUserID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE resources SET uploader_id = ? WHERE uploader_id = ?`, fallbackUserID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE issue_attachments SET created_by = ? WHERE created_by = ?`, fallbackUserID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE project_files SET created_by = ? WHERE created_by = ?`, fallbackUserID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE users SET avatar = '' WHERE id = ?`, userID); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func scanUser(row *sql.Row) (*user.User, error) {
	var u user.User
	var created string
	var adminInt int
	if err := row.Scan(&u.ID, &u.Name, &u.Account, &u.Email, &u.Avatar, &u.PasswordHash, &adminInt, &created); err != nil {
		return nil, err
	}
	if u.Account == "" {
		u.Account = u.Email
	}
	u.IsAdmin = adminInt != 0
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

func scanUserRows(rows *sql.Rows) (*user.User, error) {
	var u user.User
	var created string
	var adminInt int
	if err := rows.Scan(&u.ID, &u.Name, &u.Account, &u.Email, &u.Avatar, &u.PasswordHash, &adminInt, &created); err != nil {
		return nil, err
	}
	if u.Account == "" {
		u.Account = u.Email
	}
	u.IsAdmin = adminInt != 0
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

// --- projects ---

func (s *Store) CreateProject(p *project.Project) error {
	p.ID = uuid.NewString()
	p.CreatedAt = time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		`INSERT INTO projects(id, `+s.projectKeyCol("")+`, name, description, creator_id, created_at) VALUES(?,?,?,?,?,?)`,
		p.ID, p.Key, p.Name, p.Description, p.CreatorID, p.CreatedAt.Format(time.RFC3339),
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO project_members(project_id, user_id, role, joined_at) VALUES(?,?,?,?)`,
		p.ID, p.CreatorID, member.RoleOwner, p.CreatedAt.Format(time.RFC3339),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListProjects(userID string) ([]project.Project, error) {
	pk := s.projectKeyCol("p")
	rows, err := s.db.Query(`
SELECT p.id, `+pk+`, p.name, p.description, p.creator_id, p.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.project_id = p.id) AS issue_count,
       m.role
FROM projects p
INNER JOIN project_members m ON m.project_id = p.id AND m.user_id = ?
ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []project.Project
	for rows.Next() {
		var p project.Project
		var created string
		if err := rows.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatorID, &created, &p.IssueCount, &p.MyRole); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProject(id string) (*project.Project, error) {
	pk := s.projectKeyCol("p")
	row := s.db.QueryRow(`
SELECT p.id, `+pk+`, p.name, p.description, p.creator_id, p.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.project_id = p.id) AS issue_count
FROM projects p WHERE p.id = ?`, id)
	var p project.Project
	var created string
	if err := row.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatorID, &created, &p.IssueCount); err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) GetProjectByKey(key string) (*project.Project, error) {
	pk := s.projectKeyCol("p")
	row := s.db.QueryRow(`
SELECT p.id, `+pk+`, p.name, p.description, p.creator_id, p.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.project_id = p.id) AS issue_count
FROM projects p WHERE `+pk+` = ?`, key)
	var p project.Project
	var created string
	if err := row.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.CreatorID, &created, &p.IssueCount); err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) UpdateProject(p *project.Project) error {
	_, err := s.db.Exec(
		`UPDATE projects SET name = ?, description = ? WHERE id = ?`,
		p.Name, p.Description, p.ID,
	)
	return err
}

// --- issues ---

// IssueFilter filters issue queries.
type IssueFilter struct {
	ProjectID  string
	Type       string
	Status     string
	AssigneeID string
	CycleID    string
	ModuleID   string
	ParentID   string
	Due        string // overdue | week | none | all | ""
	RootsOnly  bool   // true => parent_id IS NULL（看板默认）
	Backlog    bool   // true => cycle_id IS NULL
	MineOnly   bool
	UserID     string
}

func (s *Store) nextIssueNumber(projectID string) (int, error) {
	var n sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(number) FROM issues WHERE project_id = ?`, projectID).Scan(&n)
	if err != nil {
		return 0, err
	}
	if !n.Valid {
		return 1, nil
	}
	return int(n.Int64) + 1, nil
}

func (s *Store) CreateIssue(it *issue.Issue) error {
	num, err := s.nextIssueNumber(it.ProjectID)
	if err != nil {
		return err
	}
	it.ID = uuid.NewString()
	it.Number = num
	now := time.Now().UTC()
	it.CreatedAt = now
	it.UpdatedAt = now
	it.BoardOrder = float64(now.UnixMilli())
	_, err = s.db.Exec(
		`INSERT INTO issues(id, project_id, number, type, title, description, status, priority, assignee_id, creator_id, created_at, updated_at, cycle_id, module_id, parent_id, start_date, due_date, board_order)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.ID, it.ProjectID, it.Number, it.Type, it.Title, it.Description, it.Status, it.Priority, nullStr(it.AssigneeID), it.CreatorID,
		it.CreatedAt.Format(time.RFC3339), it.UpdatedAt.Format(time.RFC3339), nullStr(it.CycleID), nullStr(it.ModuleID), nullStr(it.ParentID),
		nullStr(it.StartDate), nullStr(it.DueDate), it.BoardOrder,
	)
	if err != nil {
		return err
	}
	if it.ParentID != "" {
		_ = s.syncIssueProgressFromChildren(it.ParentID)
	}
	return s.hydrateIssue(it)
}

func (s *Store) issueSelectSQL() string {
	pk := s.projectKeyCol("p")
	parentKey := s.sqlConcat(pk, "'-'", "pi.number")
	return `
SELECT i.id, i.project_id, ` + pk + `, p.name, i.number, i.type, i.title, i.description, i.status, i.priority,
       COALESCE(i.assignee_id, ''), COALESCE(u.name, ''), COALESCE(i.cycle_id, ''), COALESCE(c.name, ''),
       COALESCE(i.module_id, ''), COALESCE(m.name, ''),
       COALESCE(i.parent_id, ''),
       CASE WHEN pi.id IS NULL THEN '' ELSE ` + parentKey + ` END,
       COALESCE(i.start_date, ''), COALESCE(i.due_date, ''),
       COALESCE(i.board_order, 0),
       i.creator_id, i.created_at, i.updated_at,
       (SELECT COUNT(1) FROM issues ch WHERE ch.parent_id = i.id),
       (SELECT COUNT(1) FROM issues ch WHERE ch.parent_id = i.id AND ch.status = 'done'),
       (SELECT COUNT(1) FROM issue_attachments a WHERE a.issue_id = i.id)
FROM issues i
JOIN projects p ON p.id = i.project_id
LEFT JOIN users u ON u.id = i.assignee_id
LEFT JOIN cycles c ON c.id = i.cycle_id
LEFT JOIN modules m ON m.id = i.module_id
LEFT JOIN issues pi ON pi.id = i.parent_id`
}

func (s *Store) ListIssues(f IssueFilter) ([]issue.Issue, error) {
	q := s.issueSelectSQL() + `
WHERE 1=1`
	args := []any{}
	if f.UserID != "" {
		q += ` AND EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = i.project_id AND pm.user_id = ?)`
		args = append(args, f.UserID)
	}
	if f.ProjectID != "" {
		q += ` AND i.project_id = ?`
		args = append(args, f.ProjectID)
	}
	if f.Type != "" {
		q += ` AND i.type = ?`
		args = append(args, f.Type)
	}
	if f.Status != "" {
		q += ` AND i.status = ?`
		args = append(args, f.Status)
	}
	if f.AssigneeID != "" {
		q += ` AND i.assignee_id = ?`
		args = append(args, f.AssigneeID)
	}
	if f.CycleID != "" {
		q += ` AND i.cycle_id = ?`
		args = append(args, f.CycleID)
	}
	if f.ModuleID != "" {
		q += ` AND i.module_id = ?`
		args = append(args, f.ModuleID)
	}
	if f.ParentID != "" {
		q += ` AND i.parent_id = ?`
		args = append(args, f.ParentID)
	}
	if f.RootsOnly {
		q += ` AND (i.parent_id IS NULL OR i.parent_id = '')`
	}
	if f.Backlog {
		q += ` AND (i.cycle_id IS NULL OR i.cycle_id = '')`
	}
	if f.MineOnly && f.UserID != "" {
		q += ` AND i.assignee_id = ?`
		args = append(args, f.UserID)
	}
	today := time.Now().UTC().Format("2006-01-02")
	switch strings.ToLower(strings.TrimSpace(f.Due)) {
	case "overdue":
		q += ` AND i.due_date IS NOT NULL AND i.due_date != '' AND i.due_date < ? AND i.status != ?`
		args = append(args, today, issue.StatusDone)
	case "week":
		end := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
		q += ` AND i.due_date IS NOT NULL AND i.due_date != '' AND i.due_date >= ? AND i.due_date <= ?`
		args = append(args, today, end)
	case "none":
		q += ` AND (i.due_date IS NULL OR i.due_date = '')`
	}
	q += ` ORDER BY i.board_order DESC, i.updated_at DESC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []issue.Issue
	for rows.Next() {
		it, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

func (s *Store) GetIssue(id string) (*issue.Issue, error) {
	row := s.db.QueryRow(s.issueSelectSQL()+` WHERE i.id = ?`, id)
	return scanIssueRow(row)
}

func (s *Store) UpdateIssue(it *issue.Issue) error {
	it.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(
		`UPDATE issues SET type=?, title=?, description=?, status=?, priority=?, assignee_id=?, cycle_id=?, module_id=?, start_date=?, due_date=?, updated_at=? WHERE id=?`,
		it.Type, it.Title, it.Description, it.Status, it.Priority, nullStr(it.AssigneeID), nullStr(it.CycleID), nullStr(it.ModuleID),
		nullStr(it.StartDate), nullStr(it.DueDate), it.UpdatedAt.Format(time.RFC3339), it.ID,
	)
	if err != nil {
		return err
	}
	_ = s.SyncParentFromChildren(it.ID)
	return s.hydrateIssue(it)
}

func (s *Store) UpdateIssueStatus(id string, status issue.Status) (*issue.Issue, error) {
	now := time.Now().UTC()
	// 改状态且未指定落点：按最后修改沉到该泳道顶部
	_, err := s.db.Exec(
		`UPDATE issues SET status=?, board_order=?, updated_at=? WHERE id=?`,
		status, float64(now.UnixMilli()), now.Format(time.RFC3339), id,
	)
	if err != nil {
		return nil, err
	}
	_ = s.SyncParentFromChildren(id)
	return s.GetIssue(id)
}

const boardOrderGap = 1000.0

// ReorderIssueBoard moves an issue into a status lane at a relative position.
// beforeID is the card that should sit above; afterID is the card below.
// When both empty, the card goes to the top of the lane (last-modified order).
func (s *Store) ReorderIssueBoard(id string, status issue.Status, beforeID, afterID string) (*issue.Issue, error) {
	current, err := s.GetIssue(id)
	if err != nil {
		return nil, err
	}
	order, err := s.computeBoardOrder(current.ProjectID, status, id, beforeID, afterID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(
		`UPDATE issues SET status=?, board_order=?, updated_at=? WHERE id=?`,
		status, order, now.Format(time.RFC3339), id,
	)
	if err != nil {
		return nil, err
	}
	_ = s.SyncParentFromChildren(id)
	return s.GetIssue(id)
}

func (s *Store) computeBoardOrder(projectID string, status issue.Status, movingID, beforeID, afterID string) (float64, error) {
	var beforeOrder, afterOrder float64
	hasBefore, hasAfter := false, false
	if beforeID != "" && beforeID != movingID {
		it, err := s.GetIssue(beforeID)
		if err != nil {
			return 0, err
		}
		if it.ProjectID != projectID {
			return 0, fmt.Errorf("before issue project mismatch")
		}
		beforeOrder = it.BoardOrder
		hasBefore = true
	}
	if afterID != "" && afterID != movingID {
		it, err := s.GetIssue(afterID)
		if err != nil {
			return 0, err
		}
		if it.ProjectID != projectID {
			return 0, fmt.Errorf("after issue project mismatch")
		}
		afterOrder = it.BoardOrder
		hasAfter = true
	}
	switch {
	case hasBefore && hasAfter:
		mid := (beforeOrder + afterOrder) / 2
		if mid != beforeOrder && mid != afterOrder {
			return mid, nil
		}
		// 间距耗尽时重新铺开该泳道
		if err := s.resequenceBoardLane(projectID, status, movingID); err != nil {
			return 0, err
		}
		return s.computeBoardOrder(projectID, status, movingID, beforeID, afterID)
	case hasBefore:
		return beforeOrder - boardOrderGap, nil
	case hasAfter:
		return afterOrder + boardOrderGap, nil
	default:
		return float64(time.Now().UTC().UnixMilli()), nil
	}
}

func (s *Store) resequenceBoardLane(projectID string, status issue.Status, movingID string) error {
	rows, err := s.db.Query(
		`SELECT id FROM issues WHERE project_id=? AND status=? AND id!=? ORDER BY board_order DESC, updated_at DESC`,
		projectID, status, movingID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	base := float64(time.Now().UTC().UnixMilli())
	for i, id := range ids {
		order := base - float64(i+1)*boardOrderGap
		if _, err := s.db.Exec(`UPDATE issues SET board_order=? WHERE id=?`, order, id); err != nil {
			return err
		}
	}
	return nil
}

// SyncParentFromChildren rolls parent status from children when the given issue is a child.
func (s *Store) SyncParentFromChildren(childID string) error {
	var parentID sql.NullString
	err := s.db.QueryRow(`SELECT parent_id FROM issues WHERE id = ?`, childID).Scan(&parentID)
	if err != nil {
		return err
	}
	if !parentID.Valid || parentID.String == "" {
		return nil
	}
	return s.syncIssueProgressFromChildren(parentID.String)
}

func desiredParentStatus(total, done, active int, current issue.Status) (issue.Status, bool) {
	if total == 0 {
		return current, false
	}
	if done == total {
		if current == issue.StatusDone || current == issue.StatusReview {
			return current, false
		}
		return issue.StatusReview, true
	}
	if done > 0 || active > 0 {
		switch current {
		case issue.StatusTodo, issue.StatusReview, issue.StatusDone:
			return issue.StatusDoing, true
		default:
			return current, false
		}
	}
	if current == issue.StatusDoing || current == issue.StatusReview {
		return issue.StatusTodo, true
	}
	return current, false
}

func (s *Store) syncIssueProgressFromChildren(parentID string) error {
	var total, done, active int
	err := s.db.QueryRow(`
		SELECT COUNT(1),
		       COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status IN ('doing', 'review') THEN 1 ELSE 0 END), 0)
		FROM issues WHERE parent_id = ?`, parentID).Scan(&total, &done, &active)
	if err != nil {
		return err
	}
	parent, err := s.GetIssue(parentID)
	if err != nil {
		return err
	}
	next, changed := desiredParentStatus(total, done, active, parent.Status)
	if !changed {
		return nil
	}
	_, err = s.db.Exec(
		`UPDATE issues SET status=?, updated_at=? WHERE id=?`,
		next, time.Now().UTC().Format(time.RFC3339), parentID,
	)
	return err
}

func (s *Store) hydrateIssue(it *issue.Issue) error {
	full, err := s.GetIssue(it.ID)
	if err != nil {
		return err
	}
	*it = *full
	return nil
}

func scanIssue(rows *sql.Rows) (*issue.Issue, error) {
	var it issue.Issue
	var created, updated string
	if err := rows.Scan(
		&it.ID, &it.ProjectID, &it.ProjectKey, &it.ProjectName, &it.Number, &it.Type, &it.Title, &it.Description,
		&it.Status, &it.Priority, &it.AssigneeID, &it.AssigneeName, &it.CycleID, &it.CycleName,
		&it.ModuleID, &it.ModuleName, &it.ParentID, &it.ParentKey, &it.StartDate, &it.DueDate,
		&it.BoardOrder, &it.CreatorID, &created, &updated,
		&it.ChildCount, &it.ChildDone, &it.AttachmentCount,
	); err != nil {
		return nil, err
	}
	it.Key = fmt.Sprintf("%s-%d", it.ProjectKey, it.Number)
	it.CreatedAt, _ = time.Parse(time.RFC3339, created)
	it.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &it, nil
}

func scanIssueRow(row *sql.Row) (*issue.Issue, error) {
	var it issue.Issue
	var created, updated string
	if err := row.Scan(
		&it.ID, &it.ProjectID, &it.ProjectKey, &it.ProjectName, &it.Number, &it.Type, &it.Title, &it.Description,
		&it.Status, &it.Priority, &it.AssigneeID, &it.AssigneeName, &it.CycleID, &it.CycleName,
		&it.ModuleID, &it.ModuleName, &it.ParentID, &it.ParentKey, &it.StartDate, &it.DueDate,
		&it.BoardOrder, &it.CreatorID, &created, &updated,
		&it.ChildCount, &it.ChildDone, &it.AttachmentCount,
	); err != nil {
		return nil, err
	}
	it.Key = fmt.Sprintf("%s-%d", it.ProjectKey, it.Number)
	it.CreatedAt, _ = time.Parse(time.RFC3339, created)
	it.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &it, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// --- comments ---

func (s *Store) ListComments(issueID string) ([]comment.Comment, error) {
	rows, err := s.db.Query(`
SELECT c.id, c.issue_id, c.author_id, COALESCE(u.name, ''),
  COALESCE(NULLIF(u.account,''), COALESCE(u.email,'')), COALESCE(u.avatar,''),
  c.body, c.created_at
FROM comments c
LEFT JOIN users u ON u.id = c.author_id
WHERE c.issue_id = ?
ORDER BY c.created_at ASC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []comment.Comment
	for rows.Next() {
		var cm comment.Comment
		var created, avatar string
		if err := rows.Scan(&cm.ID, &cm.IssueID, &cm.AuthorID, &cm.AuthorName, &cm.AuthorAccount, &avatar, &cm.Body, &created); err != nil {
			return nil, err
		}
		if strings.TrimSpace(avatar) != "" {
			cm.AuthorAvatarURL = "/api/avatars/" + cm.AuthorID
		}
		cm.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, cm)
	}
	return out, rows.Err()
}

func (s *Store) CreateComment(issueID, authorID, body string) (*comment.Comment, error) {
	cm := &comment.Comment{
		ID:        uuid.NewString(),
		IssueID:   issueID,
		AuthorID:  authorID,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	if _, err := s.db.Exec(
		`INSERT INTO comments(id, issue_id, author_id, body, created_at) VALUES(?,?,?,?,?)`,
		cm.ID, cm.IssueID, cm.AuthorID, cm.Body, cm.CreatedAt.Format(time.RFC3339),
	); err != nil {
		return nil, err
	}
	u, err := s.GetUserByID(authorID)
	if err == nil {
		cm.AuthorName = u.Name
		cm.AuthorAccount = u.Account
		if cm.AuthorAccount == "" {
			cm.AuthorAccount = u.Email
		}
		cm.AuthorAvatarURL = u.AvatarURL()
	}
	_, _ = s.db.Exec(`UPDATE issues SET updated_at=? WHERE id=?`, cm.CreatedAt.Format(time.RFC3339), issueID)
	return cm, nil
}

// --- members ---

func (s *Store) ListMembers(projectID string) ([]member.Member, error) {
	rows, err := s.db.Query(`
SELECT m.project_id, m.user_id, u.name, COALESCE(NULLIF(u.account,''), u.email), COALESCE(u.email,''), COALESCE(u.avatar,''), m.role, m.joined_at
FROM project_members m
JOIN users u ON u.id = m.user_id
WHERE m.project_id = ?
ORDER BY
  CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'member' THEN 2 ELSE 3 END,
  m.joined_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []member.Member
	for rows.Next() {
		var m member.Member
		var joined, avatar string
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Name, &m.Account, &m.Email, &avatar, &m.Role, &joined); err != nil {
			return nil, err
		}
		if m.Email == m.Account {
			m.Email = ""
		}
		if avatar != "" {
			m.AvatarURL = "/api/avatars/" + m.UserID
		}
		m.JoinedAt, _ = time.Parse(time.RFC3339, joined)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMember(projectID, userID string) (*member.Member, error) {
	row := s.db.QueryRow(`
SELECT m.project_id, m.user_id, u.name, COALESCE(NULLIF(u.account,''), u.email), COALESCE(u.email,''), COALESCE(u.avatar,''), m.role, m.joined_at
FROM project_members m
JOIN users u ON u.id = m.user_id
WHERE m.project_id = ? AND m.user_id = ?`, projectID, userID)
	var m member.Member
	var joined, avatar string
	if err := row.Scan(&m.ProjectID, &m.UserID, &m.Name, &m.Account, &m.Email, &avatar, &m.Role, &joined); err != nil {
		return nil, err
	}
	if m.Email == m.Account {
		m.Email = ""
	}
	if avatar != "" {
		m.AvatarURL = "/api/avatars/" + m.UserID
	}
	m.JoinedAt, _ = time.Parse(time.RFC3339, joined)
	return &m, nil
}

func (s *Store) AddMember(projectID, userID string, role member.Role) (*member.Member, error) {
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`INSERT INTO project_members(project_id, user_id, role, joined_at) VALUES(?,?,?,?)`,
		projectID, userID, role, now.Format(time.RFC3339),
	); err != nil {
		return nil, err
	}
	return s.GetMember(projectID, userID)
}

func (s *Store) UpdateMemberRole(projectID, userID string, role member.Role) (*member.Member, error) {
	res, err := s.db.Exec(`UPDATE project_members SET role=? WHERE project_id=? AND user_id=?`, role, projectID, userID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	return s.GetMember(projectID, userID)
}

func (s *Store) RemoveMember(projectID, userID string) error {
	res, err := s.db.Exec(`DELETE FROM project_members WHERE project_id=? AND user_id=?`, projectID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) CountOwners(projectID string) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(1) FROM project_members WHERE project_id=? AND role=?`,
		projectID, member.RoleOwner,
	).Scan(&n)
	return n, err
}

// --- cycles ---

func (s *Store) ListCycles(projectID string) ([]cycle.Cycle, error) {
	rows, err := s.db.Query(`
SELECT c.id, c.project_id, c.name, c.goal, c.status, c.start_date, c.end_date, c.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.cycle_id = c.id) AS issue_count,
       (SELECT COUNT(1) FROM issues i WHERE i.cycle_id = c.id AND i.status = 'done') AS done_count
FROM cycles c
WHERE c.project_id = ?
ORDER BY
  CASE c.status WHEN 'active' THEN 0 WHEN 'planned' THEN 1 ELSE 2 END,
  c.start_date DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []cycle.Cycle
	for rows.Next() {
		cy, err := scanCycle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *cy)
	}
	return out, rows.Err()
}

func (s *Store) GetCycle(id string) (*cycle.Cycle, error) {
	row := s.db.QueryRow(`
SELECT c.id, c.project_id, c.name, c.goal, c.status, c.start_date, c.end_date, c.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.cycle_id = c.id) AS issue_count,
       (SELECT COUNT(1) FROM issues i WHERE i.cycle_id = c.id AND i.status = 'done') AS done_count
FROM cycles c WHERE c.id = ?`, id)
	return scanCycleRow(row)
}

func (s *Store) CreateCycle(cy *cycle.Cycle) error {
	cy.ID = uuid.NewString()
	cy.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(
		`INSERT INTO cycles(id, project_id, name, goal, status, start_date, end_date, created_at) VALUES(?,?,?,?,?,?,?,?)`,
		cy.ID, cy.ProjectID, cy.Name, cy.Goal, cy.Status, cy.StartDate, cy.EndDate, cy.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	full, err := s.GetCycle(cy.ID)
	if err != nil {
		return err
	}
	*cy = *full
	return nil
}

func (s *Store) UpdateCycle(cy *cycle.Cycle) error {
	_, err := s.db.Exec(
		`UPDATE cycles SET name=?, goal=?, status=?, start_date=?, end_date=? WHERE id=?`,
		cy.Name, cy.Goal, cy.Status, cy.StartDate, cy.EndDate, cy.ID,
	)
	if err != nil {
		return err
	}
	full, err := s.GetCycle(cy.ID)
	if err != nil {
		return err
	}
	*cy = *full
	return nil
}

func scanCycle(rows *sql.Rows) (*cycle.Cycle, error) {
	var cy cycle.Cycle
	var created string
	if err := rows.Scan(
		&cy.ID, &cy.ProjectID, &cy.Name, &cy.Goal, &cy.Status, &cy.StartDate, &cy.EndDate, &created,
		&cy.IssueCount, &cy.DoneCount,
	); err != nil {
		return nil, err
	}
	cy.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &cy, nil
}

func scanCycleRow(row *sql.Row) (*cycle.Cycle, error) {
	var cy cycle.Cycle
	var created string
	if err := row.Scan(
		&cy.ID, &cy.ProjectID, &cy.Name, &cy.Goal, &cy.Status, &cy.StartDate, &cy.EndDate, &created,
		&cy.IssueCount, &cy.DoneCount,
	); err != nil {
		return nil, err
	}
	cy.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &cy, nil
}

// --- modules ---

func (s *Store) ListModules(projectID string) ([]module.Module, error) {
	rows, err := s.db.Query(`
SELECT m.id, m.project_id, m.name, m.description, m.status, COALESCE(m.lead_id, ''), COALESCE(u.name, ''),
       m.start_date, m.target_date, m.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.module_id = m.id) AS issue_count,
       (SELECT COUNT(1) FROM issues i WHERE i.module_id = m.id AND i.status = 'done') AS done_count
FROM modules m
LEFT JOIN users u ON u.id = m.lead_id
WHERE m.project_id = ?
` + s.orderByNameAsc("m.name"), projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []module.Module
	for rows.Next() {
		md, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *md)
	}
	return out, rows.Err()
}

func (s *Store) GetModule(id string) (*module.Module, error) {
	row := s.db.QueryRow(`
SELECT m.id, m.project_id, m.name, m.description, m.status, COALESCE(m.lead_id, ''), COALESCE(u.name, ''),
       m.start_date, m.target_date, m.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.module_id = m.id) AS issue_count,
       (SELECT COUNT(1) FROM issues i WHERE i.module_id = m.id AND i.status = 'done') AS done_count
FROM modules m
LEFT JOIN users u ON u.id = m.lead_id
WHERE m.id = ?`, id)
	return scanModuleRow(row)
}

func (s *Store) FindModuleByName(projectID, name string) (*module.Module, error) {
	row := s.db.QueryRow(`
SELECT m.id, m.project_id, m.name, m.description, m.status, COALESCE(m.lead_id, ''), COALESCE(u.name, ''),
       m.start_date, m.target_date, m.created_at,
       (SELECT COUNT(1) FROM issues i WHERE i.module_id = m.id) AS issue_count,
       (SELECT COUNT(1) FROM issues i WHERE i.module_id = m.id AND i.status = 'done') AS done_count
FROM modules m
LEFT JOIN users u ON u.id = m.lead_id
WHERE m.project_id = ? AND m.name = ?`, projectID, name)
	return scanModuleRow(row)
}

// EnsureModule finds or creates a tag-like module by name within a project.
func (s *Store) EnsureModule(projectID, name string) (*module.Module, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, sql.ErrNoRows
	}
	md, err := s.FindModuleByName(projectID, name)
	if err == nil {
		return md, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	md = &module.Module{
		ProjectID: projectID,
		Name:      name,
		Status:    module.StatusActive,
	}
	if err := s.CreateModule(md); err != nil {
		return nil, err
	}
	return md, nil
}

func (s *Store) CreateModule(md *module.Module) error {
	md.ID = uuid.NewString()
	md.CreatedAt = time.Now().UTC()
	if md.Status == "" {
		md.Status = module.StatusActive
	}
	_, err := s.db.Exec(
		`INSERT INTO modules(id, project_id, name, description, status, lead_id, start_date, target_date, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		md.ID, md.ProjectID, md.Name, md.Description, md.Status, nullStr(md.LeadID), md.StartDate, md.TargetDate,
		md.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	full, err := s.GetModule(md.ID)
	if err != nil {
		return err
	}
	*md = *full
	return nil
}

func (s *Store) UpdateModule(md *module.Module) error {
	_, err := s.db.Exec(
		`UPDATE modules SET name=?, description=?, status=?, lead_id=?, start_date=?, target_date=? WHERE id=?`,
		md.Name, md.Description, md.Status, nullStr(md.LeadID), md.StartDate, md.TargetDate, md.ID,
	)
	if err != nil {
		return err
	}
	full, err := s.GetModule(md.ID)
	if err != nil {
		return err
	}
	*md = *full
	return nil
}

// DeleteModule clears issue links then removes the module tag.
func (s *Store) DeleteModule(id string) error {
	if _, err := s.db.Exec(`UPDATE issues SET module_id = NULL WHERE module_id = ?`, id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM modules WHERE id = ?`, id)
	return err
}

func scanModule(rows *sql.Rows) (*module.Module, error) {
	var md module.Module
	var created string
	if err := rows.Scan(
		&md.ID, &md.ProjectID, &md.Name, &md.Description, &md.Status, &md.LeadID, &md.LeadName,
		&md.StartDate, &md.TargetDate, &created, &md.IssueCount, &md.DoneCount,
	); err != nil {
		return nil, err
	}
	md.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &md, nil
}

func scanModuleRow(row *sql.Row) (*module.Module, error) {
	var md module.Module
	var created string
	if err := row.Scan(
		&md.ID, &md.ProjectID, &md.Name, &md.Description, &md.Status, &md.LeadID, &md.LeadName,
		&md.StartDate, &md.TargetDate, &created, &md.IssueCount, &md.DoneCount,
	); err != nil {
		return nil, err
	}
	md.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &md, nil
}

// --- notifications ---

// CreateNotification inserts an in-app notification for a user.
// Callers decide whether self-actions should notify (e.g. assign-to-self yes, self-comment no).
func (s *Store) CreateNotification(userID, actorID string, kind notification.Kind, it *issue.Issue, body string) error {
	if userID == "" || actorID == "" || it == nil || it.ID == "" {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO notifications(id, user_id, actor_id, kind, issue_id, project_id, body, read_at, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		uuid.NewString(), userID, actorID, kind, it.ID, it.ProjectID, strings.TrimSpace(body), nil,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// ListNotifications returns recent notifications for a user (newest first).
func (s *Store) ListNotifications(userID string, limit int) ([]notification.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	issueKey := s.sqlConcat(s.projectKeyCol("p"), "'-'", "i.number")
	rows, err := s.db.Query(`
SELECT n.id, n.user_id, n.actor_id, COALESCE(a.name, ''), n.kind, n.issue_id,
       `+issueKey+`, i.title, n.project_id, p.name, n.body, n.read_at, n.created_at
FROM notifications n
JOIN users a ON a.id = n.actor_id
JOIN issues i ON i.id = n.issue_id
JOIN projects p ON p.id = n.project_id
WHERE n.user_id = ?
ORDER BY n.created_at DESC
LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []notification.Notification
	for rows.Next() {
		var n notification.Notification
		var readAt sql.NullString
		var created string
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.ActorID, &n.ActorName, &n.Kind, &n.IssueID,
			&n.IssueKey, &n.IssueTitle, &n.ProjectID, &n.ProjectName, &n.Body, &readAt, &created,
		); err != nil {
			return nil, err
		}
		if readAt.Valid && readAt.String != "" {
			t, err := time.Parse(time.RFC3339, readAt.String)
			if err == nil {
				n.ReadAt = &t
			}
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountUnreadNotifications returns unread count for a user.
func (s *Store) CountUnreadNotifications(userID string) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(1) FROM notifications WHERE user_id = ? AND read_at IS NULL`,
		userID,
	).Scan(&n)
	return n, err
}

// MarkNotificationRead marks one notification as read for its owner.
func (s *Store) MarkNotificationRead(id, userID string) error {
	_, err := s.db.Exec(
		`UPDATE notifications SET read_at = ? WHERE id = ? AND user_id = ? AND read_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), id, userID,
	)
	return err
}

// MarkAllNotificationsRead marks all unread notifications for a user.
func (s *Store) MarkAllNotificationsRead(userID string) error {
	_, err := s.db.Exec(
		`UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), userID,
	)
	return err
}

// ensureDemoNotifications seeds a couple of unread items for the demo PM.
func (s *Store) ensureDemoNotifications() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM notifications`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var adminID, pmID, projectID, issueID string
	err := s.db.QueryRow(`SELECT id FROM users WHERE email = 'admin@ningpm.local'`).Scan(&adminID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.db.QueryRow(`SELECT id FROM users WHERE email = 'pm@ningpm.local'`).Scan(&pmID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.db.QueryRow(`SELECT id FROM projects WHERE ` + s.projectKeyCol("") + ` = 'DEMO'`).Scan(&projectID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.db.QueryRow(
		`SELECT id FROM issues WHERE project_id = ? AND assignee_id = ? ORDER BY number LIMIT 1`,
		projectID, pmID,
	).Scan(&issueID)
	if err == sql.ErrNoRows {
		err = s.db.QueryRow(
			`SELECT id FROM issues WHERE project_id = ? ORDER BY number LIMIT 1`,
			projectID,
		).Scan(&issueID)
	}
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	it, err := s.GetIssue(issueID)
	if err != nil {
		return err
	}
	if err := s.CreateNotification(pmID, adminID, notification.KindAssigned, it, ""); err != nil {
		return err
	}
	if err := s.CreateNotification(pmID, adminID, notification.KindCommented, it, "先看一下这条，排进本周？"); err != nil {
		return err
	}
	// Default login is admin — leave an unread item so the bell lights up immediately.
	return s.CreateNotification(adminID, pmID, notification.KindCommented, it, "我先跟一下这条，有进展再同步。")
}

// ensureDemoSubIssues seeds one-level children under a DEMO story/task.
func (s *Store) ensureDemoSubIssues() error {
	var projectID, adminID, parentID string
	err := s.db.QueryRow(`SELECT id, creator_id FROM projects WHERE ` + s.projectKeyCol("") + ` = 'DEMO'`).Scan(&projectID, &adminID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var n int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM issues WHERE project_id = ? AND parent_id IS NOT NULL AND parent_id != ''`,
		projectID,
	).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	err = s.db.QueryRow(
		`SELECT id FROM issues WHERE project_id = ? AND type = ? AND (parent_id IS NULL OR parent_id = '') ORDER BY number LIMIT 1`,
		projectID, issue.TypeStory,
	).Scan(&parentID)
	if err == sql.ErrNoRows {
		err = s.db.QueryRow(
			`SELECT id FROM issues WHERE project_id = ? AND (parent_id IS NULL OR parent_id = '') ORDER BY number LIMIT 1`,
			projectID,
		).Scan(&parentID)
	}
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	parent, err := s.GetIssue(parentID)
	if err != nil {
		return err
	}
	samples := []struct {
		typ    issue.Type
		title  string
		status issue.Status
	}{
		{issue.TypeTask, "拆解：接口契约与字段对齐", issue.StatusDoing},
		{issue.TypeTask, "拆解：前端列表联调", issue.StatusTodo},
		{issue.TypeBug, "拆解：边界空数据回归", issue.StatusTodo},
	}
	for _, sample := range samples {
		child := &issue.Issue{
			ProjectID:  projectID,
			Type:       sample.typ,
			Title:      sample.title,
			Status:     sample.status,
			Priority:   parent.Priority,
			AssigneeID: parent.AssigneeID,
			CycleID:    parent.CycleID,
			ModuleID:   parent.ModuleID,
			ParentID:   parent.ID,
			CreatorID:  adminID,
		}
		if err := s.CreateIssue(child); err != nil {
			return err
		}
	}
	return nil
}
