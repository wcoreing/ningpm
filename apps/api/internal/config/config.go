package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config holds process-level settings.
type Config struct {
	HTTPAddr  string
	Env       string
	DBDriver  string // sqlite | mysql
	DBPath    string // sqlite file path
	DBDSN     string // mysql DSN
	FilesDir  string
	JWTSecret string
	SeedAdmin bool
}

// Load reads configuration from environment.
func Load() Config {
	addr := strings.TrimSpace(os.Getenv("NINGPM_HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	env := strings.TrimSpace(os.Getenv("NINGPM_ENV"))
	if env == "" {
		env = "development"
	}
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("NINGPM_DB_DRIVER")))
	if driver == "" {
		driver = "sqlite"
	}
	dbPath := strings.TrimSpace(os.Getenv("NINGPM_DB_PATH"))
	if dbPath == "" {
		dbPath = filepath.Join("data", "ningpm.db")
	}
	dsn := strings.TrimSpace(os.Getenv("NINGPM_DB_DSN"))
	filesDir := strings.TrimSpace(os.Getenv("NINGPM_FILES_DIR"))
	if filesDir == "" {
		if driver == "mysql" {
			filesDir = filepath.Join("data", "files")
		} else {
			filesDir = filepath.Join(filepath.Dir(dbPath), "files")
		}
	}
	secret := strings.TrimSpace(os.Getenv("NINGPM_JWT_SECRET"))
	if secret == "" {
		secret = "ningpm-dev-secret-change-me"
	}
	return Config{
		HTTPAddr:  addr,
		Env:       env,
		DBDriver:  driver,
		DBPath:    dbPath,
		DBDSN:     dsn,
		FilesDir:  filesDir,
		JWTSecret: secret,
		SeedAdmin: true,
	}
}
