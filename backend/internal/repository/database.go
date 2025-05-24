package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trasta298/kasaneha/backend/internal/config"
)

// Database wraps the database connection pool
type Database struct {
	Pool *pgxpool.Pool
}

// NewDatabase creates a new database instance
func NewDatabase(cfg *config.Config) (*Database, error) {
	// Parse connection config
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set connection pool settings
	poolConfig.MaxConns = int32(cfg.Database.MaxConnections)
	poolConfig.MinConns = int32(cfg.Database.MinConnections)

	// Parse durations
	if maxLifetime, err := time.ParseDuration(cfg.Database.MaxConnLifetime); err == nil {
		poolConfig.MaxConnLifetime = maxLifetime
	}
	if maxIdleTime, err := time.ParseDuration(cfg.Database.MaxConnIdleTime); err == nil {
		poolConfig.MaxConnIdleTime = maxIdleTime
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *Database) Close() {
	db.Pool.Close()
}

// Health checks if the database is healthy
func (db *Database) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
