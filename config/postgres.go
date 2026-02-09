package config

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewPostgres(cfg *Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return db, nil
}
