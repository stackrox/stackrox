package dbs

import (
	"context"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

// NewPostgresSize returns a generator for Postgres backups.
// We take in the connection to connect to the DB
func NewPostgresSize(db postgres.DB) *PostgresSize {
	return &PostgresSize{
		db: db,
	}
}

// PostgresSize is an implementation of a StreamGenerator which writes a backup of PostgresDB to the input io.Writer.
type PostgresSize struct {
	db postgres.DB
}

// WriteTo writes a backup of Postgres to the writer
func (ps *PostgresSize) WriteTo(_ context.Context, out io.Writer) error {
	_, config, err := pgconfig.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
		return err
	}

	size, err := pgadmin.GetDatabaseSize(config, migrations.GetCurrentClone())
	if err != nil {
		return err
	}

	_, err = out.Write([]byte(fmt.Sprintf("%d", size)))
	if err != nil {
		return err
	}

	return nil
}
