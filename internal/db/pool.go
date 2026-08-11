// Package db owns the PostgreSQL connection pool.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §2.3, §13.5.
//
// pgx v5 replaces psycopg2's ThreadedConnectionPool. Pool bounds match the
// Python (min 2, max 20) so the database sees the same connection pressure.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

// ErrNotConfigured mirrors the Python's "DATABASE_URL environment variable is
// not set." A missing database must not prevent startup: /health and the
// analysis routes work without it, exactly as they do today when the Python
// logs "[DB] PostgreSQL pool init skipped ... (GEE-only mode)".
var ErrNotConfigured = errors.New("DATABASE_URL environment variable is not set")

// Pool wraps pgxpool with the project's configuration.
type Pool struct {
	*pgxpool.Pool
}

// New opens the pool and verifies connectivity.
func New(ctx context.Context, cfg *config.Config) (*Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, ErrNotConfigured
	}

	pcfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing DATABASE_URL: %w", err)
	}
	pcfg.MinConns = cfg.DBMinConns
	pcfg.MaxConns = cfg.DBMaxConns
	pcfg.MaxConnLifetime = time.Hour
	pcfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to PostgreSQL: %w", err)
	}
	return &Pool{Pool: pool}, nil
}
