package psql_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.soon.build/kit/psql"
)

func TestConfigDSN(t *testing.T) {
	config := psql.Config{
		User:    "user",
		Pass:    "pass",
		DBName:  "dbname",
		Host:    "host",
		SSLMode: "disable",
	}
	dsn := config.DSN()
	assert.Equal(t, "postgres://user:pass@host/dbname?sslmode=disable", dsn)
}
