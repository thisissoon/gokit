package psql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// Config contains data source name settings
type Config struct {
	User           string
	Pass           string
	DBName         string
	Host           string
	SSLMode        string
	MaxConnections int
}

// DSN returns the data source name as a string in the
// correct format
func (config Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		config.User,
		config.Pass,
		config.Host,
		config.DBName,
		config.SSLMode)
}

// Open opens a database connection pool to a database server
func Open(ctx context.Context, log zerolog.Logger, config Config) (*sql.DB, error) {
	log.Debug().Msg("opening db connection")
	db, err := sql.Open("postgres", config.DSN())
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	// https://aaronoellis.com/articles/preventing-max-connection-errors-in-go
	var maxConns = 20 // Sane default
	if config.MaxConnections != 0 {
		maxConns = config.MaxConnections
	}
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(time.Second * 10)
	return db, nil
}
