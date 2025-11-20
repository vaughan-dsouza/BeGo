package db

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// helper to read env with default
func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func Connect(dsn string) (*sqlx.DB, error) {
	// Parse DSN → pgx config struct
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: failed to parse DSN: %w", err)
	}

	// Fail fast on startup if PG is unreachable
	cfg.ConnectTimeout = 5 * time.Second

	// Create sql.DB using pgx's stdlib adapter
	sqlDB := stdlib.OpenDB(*cfg)

	// Wrap in sqlx for named queries & struct scanning
	db := sqlx.NewDb(sqlDB, "pgx")

	// ---- Connection Pool Settings ----
	maxOpen, _ := strconv.Atoi(getenv("DB_MAX_OPEN", "25"))
	maxIdle, _ := strconv.Atoi(getenv("DB_MAX_IDLE", "25"))
	lifetime, _ := strconv.Atoi(getenv("DB_MAX_LIFETIME", "300")) // seconds

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(lifetime) * time.Second)

	// ---- Connectivity Check ----
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db: failed to connect to Postgres: %w", err)
	}

	// ---- Health Check Query ----
	var tmp int
	if err := db.QueryRow("SELECT 1").Scan(&tmp); err != nil {
		return nil, fmt.Errorf("db: health check failed: %w", err)
	}

	return db, nil
}
