

//go:build sql_integration
{{- $name := .TrimmedType|lowerCamelCase }}
{{define "getterParamList"}}{{$name := .TrimmedType|lowerCamelCase}}{{range $index, $pk := .Schema.PrimaryKeys}}{{if $index}}, {{end}}{{$pk.Getter $name}}{{end}}{{end}}
{{ $boltDB := eq .Migration.MigrateFromDB "boltdb" }}
{{ $dackbox := eq .Migration.MigrateFromDB "dackbox" }}
{{ $rocksDB := or $dackbox (eq .Migration.MigrateFromDB "rocksdb") }}
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	{{ if $rocksDB}}"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"{{end}}
	{{ if $boltDB}}"github.com/stackrox/rox/pkg/bolthelper"{{end}}
	{{ if $dackbox}}"github.com/stackrox/rox/pkg/dackbox/concurrency"{{end}}
	{{ if $dackbox}}"github.com/stackrox/rox/pkg/dackbox"{{end}}
	{{ if $dackbox}}rawDackbox "github.com/stackrox/rox/pkg/dackbox/raw"{{end}}
	"github.com/stackrox/rox/pkg/env"
	{{ if $rocksDB}}"github.com/stackrox/rox/pkg/rocksdb"{{end}}
	"github.com/stackrox/rox/pkg/testutils"
	{{ if $rocksDB}}"github.com/stackrox/rox/pkg/testutils/rocksdbtest"{{end}}
	"github.com/stretchr/testify/suite"
	{{ if $rocksDB}}"github.com/tecbot/gorocksdb"{{end}}
	{{ if $boltDB}}bolt "go.etcd.io/bbolt"{{end}}
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	ctx		 context.Context

	legacyDB {{if $boltDB}}*bolt.DB{{else}}*rocksdb.RocksDB{{end}}
	postgresDB *pghelper.TestPostgres
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	var err error
	{{- if $rocksDB}}
	s.legacyDB, err = rocksdb.NewTemp(s.T().Name())
	{{- else}}
	s.legacyDB, err = bolthelper.NewTemp(s.T().Name() + ".db")
	{{- end}}
	s.NoError(err)

	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.postgresDB = pghelper.ForT(s.T(), false)
}

func (s *postgresMigrationSuite) TearDownTest() {
	{{- if $boltDB}}
	testutils.TearDownDB(s.legacyDB)
	{{- else}}
	rocksdbtest.TearDownRocksDB(s.legacyDB)
	{{- end}}
	s.postgresDB.Teardown(s.T())
}

func (s *postgresMigrationSuite) Test{{.TrimmedType}}Migration() {
	newStore := pgStore.New(s.postgresDB.DB)
	{{- if .Migration.SingletonStore}}
	legacyStore := legacy.New(s.legacyDB)
	{{- else if $dackbox}}
	dacky, err := dackbox.NewRocksDBDackBox(s.legacyDB, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	s.NoError(err)
	legacyStore := legacy.New(dacky, concurrency.NewKeyFence())
	{{- else if $rocksDB}}
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)
	{{- else}}{{/* boltDB*/}}
	legacyStore := legacy.New(s.legacyDB)
	{{- end}}

	// Prepare data and write to legacy DB
	{{- if .Migration.SingletonStore}}
	{{$name}} := &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	s.NoError(legacyStore.Upsert(s.ctx, {{$name}}))
	{{- else}}{{/* non-singleton store*/}}
	var {{$name}}s []*{{.Type}}
	for i := 0; i < 200; i++ {
		{{$name}} := &{{.Type}}{}
		s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		{{$name}}s = append({{$name}}s, {{$name}})
		{{ if $boltDB}}s.NoError(legacyStore.Upsert(s.ctx, {{$name}})){{- end -}}
	}

	{{ if $rocksDB}}s.NoError(legacyStore.UpsertMany(s.ctx, {{$name}}s)){{- end}}
	{{- end}}{{/* non-singleton store*/}}

	// Move
	s.NoError(move(s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	{{- if .Migration.SingletonStore}}
	fetched, found, err := newStore.Get(s.ctx)
	s.NoError(err)
	s.True(found)
	s.Equal({{$name}}, fetched)
	{{- else}}
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len({{$name}}s), count)
	for _, {{$name}} := range {{$name}}s {
		fetched, exists, err := newStore.Get(s.ctx, {{ template "getterParamList" $ }})
		s.NoError(err)
		s.True(exists)
		s.Equal({{$name}}, fetched)
	}
	{{- end}}
}
