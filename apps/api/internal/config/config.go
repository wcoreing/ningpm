package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config holds process-level settings.
type Config struct {
	HTTPAddr   string
	Env        string
	DBPath     string
	JWTSecret  string
	SeedAdmin  bool
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
	dbPath := strings.TrimSpace(os.Getenv("NINGPM_DB_PATH"))
	if dbPath == "" {
		dbPath = filepath.Join("data", "ningpm.db")
	}
	secret := strings.TrimSpace(os.Getenv("NINGPM_JWT_SECRET"))
	if secret == "" {
		secret = "ningpm-dev-secret-change-me"
	}
	return Config{
		HTTPAddr:  addr,
		Env:       env,
		DBPath:    dbPath,
		JWTSecret: secret,
		SeedAdmin: true,
	}
}
