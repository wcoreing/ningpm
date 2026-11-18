package main

import (
	"log"

	"github.com/ningpm/ningpm/apps/api/internal/auth"
	"github.com/ningpm/ningpm/apps/api/internal/config"
	"github.com/ningpm/ningpm/apps/api/internal/httpapi"
	"github.com/ningpm/ningpm/apps/api/internal/store"
	"github.com/ningpm/ningpm/apps/api/internal/version"
)

func main() {
	cfg := config.Load()

	st, err := store.Open(store.Options{
		Driver:     cfg.DBDriver,
		SQLitePath: cfg.DBPath,
		MySQLDSN:   cfg.DBDSN,
		FilesDir:   cfg.FilesDir,
	})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	if cfg.SeedAdmin {
		if err := st.SeedDemo(); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	jwt := auth.NewManager(cfg.JWTSecret)
	srv := httpapi.NewServer(cfg, st, jwt)
	r := srv.NewRouter()

	dbInfo := cfg.DBPath
	if cfg.DBDriver == "mysql" {
		dbInfo = "mysql"
	}
	log.Printf("ningpm api %s listening on %s (%s) driver=%s db=%s", version.Number, cfg.HTTPAddr, cfg.Env, cfg.DBDriver, dbInfo)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
