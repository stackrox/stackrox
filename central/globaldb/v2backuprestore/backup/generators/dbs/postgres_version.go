package dbs

import (
	"context"
	"io"

	versionUtils "github.com/stackrox/rox/central/version/utils"
	"github.com/stackrox/rox/pkg/postgres"
	"gopkg.in/yaml.v3"
)

// NewPostgresVersion returns a generator for Postgres backups.
// We take in the connection to connect to the DB
func NewPostgresVersion(db postgres.DB) *PostgresVersion {
	return &PostgresVersion{
		db: db,
	}
}

// PostgresVersion is an implementation of a StreamGenerator which writes information from the version table to the input io.Writer.
type PostgresVersion struct {
	db postgres.DB
}

// WriteTo writes a backup of Postgres to the writer
func (ps *PostgresVersion) WriteTo(_ context.Context, out io.Writer) error {
	version, err := versionUtils.ReadVersionPostgres(ps.db)
	if err != nil {
		log.Fatalf("Could not retrieve version data: %v", err)
		return err
	}

	bytes, err := yaml.Marshal(version)
	if err != nil {
		return err
	}

	_, err = out.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}
