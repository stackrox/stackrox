//go:build sql_integration
{{define "createTableStmtVar"}}pkgSchema.CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{- $name := .TrimmedType|lowerCamelCase }}
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}
{{define "getterParamList"}}{{$name := .TrimmedType|lowerCamelCase}}{{range $idx, $pk := .Schema.PrimaryKeys}}{{if $idx}}, {{end}}{{$pk.Getter $name}}{{end}}{{end}}
{{ $boltDB := eq .Migration.MigrateFromDB "boltdb" }}
{{ $dackbox := eq .Migration.MigrateFromDB "dackbox" }}
{{ $rocksDB := or $dackbox (eq .Migration.MigrateFromDB "rocksdb") }}

import (
	"context"
	"sort"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/postgres"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	{{if $dackbox}}rawDackbox "github.com/stackrox/rox/pkg/dackbox/raw"{{end}}
	{{if $dackbox}}"github.com/stackrox/rox/pkg/dackbox"{{end}}
	{{if $dackbox}}"github.com/stackrox/rox/pkg/concurrency"{{end}}
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	ctx         context.Context

	// LegacyDB to migrate from
	legacyDB {{if $boltDB}}*bolt.DB{{else}}*rocksdb.RocksDB{{end}}

	// PostgresDB
	pool   *pgxpool.Pool
	gormDB *gorm.DB
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	var err error
    {{- if $rocksDB}}
	s.legacyDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)
	{{- end}}
	{{- if $boltDB}}
    s.legacyDB, err = bolthelper.NewTemp(s.T().Name() + ".db")
    s.NoError(err)
	{{- end}}

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.pool, err = pgxpool.ConnectConfig(s.ctx, config)
	s.Require().NoError(err)
	pgtest.CleanUpDB(s.ctx, s.T(), s.pool)
	s.gormDB = pgtest.OpenGormDBWithDisabledConstraints(s.T(), source)
}

func (s *postgresMigrationSuite) TearDownTest() {
    {{- if $boltDB}}
    testutils.TearDownDB(s.legacyDB)
	{{- else}}
	rocksdbtest.TearDownRocksDB(s.legacyDB)
	{{- end}}
	_ = s.gormDB.Migrator().DropTable({{template "createTableStmtVar" .Schema}}.GormModel)
	pgtest.CleanUpDB(s.ctx, s.T(), s.pool)
	pgtest.CloseGormDB(s.T(), s.gormDB)
    s.pool.Close()
}

func (s *postgresMigrationSuite) TestMigration() {
	newStore := pgStore.New({{if .Migration.SingletonStore}}s.ctx, {{end}}s.pool)
	// Prepare data and write to legacy DB
    {{- if .Migration.SingletonStore}}
    legacyStore := legacy.New(s.legacyDB)
    {{$name}} := &{{.Type}}{}
    s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
    s.NoError(legacyStore.Upsert(s.ctx, {{$name}}))
    // Move
    s.NoError(move(s.gormDB, s.pool, legacyStore))
    // Verify
    fetched, found, err := newStore.Get(s.ctx)
    s.NoError(err)
    s.True(found)
    s.Equal({{$name}}, fetched)
    {{- else}}
	var {{$name}}s []*{{.Type}}
	{{- if $rocksDB}}
        {{- if $dackbox}}
        dacky, err := dackbox.NewRocksDBDackBox(s.legacyDB, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
        s.NoError(err)
        legacyStore := legacy.New(dacky, concurrency.NewKeyFence())
        {{- else}}
	    legacyStore, err := legacy.New(s.legacyDB)
	    s.NoError(err)
	    {{- end}}
	{{- end}}
	{{- if $boltDB}}
	legacyStore := legacy.New(s.legacyDB)
	{{- end}}
	{{- if $rocksDB}}
	batchSize = 48
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	{{- end}}
	for i := 0; i < 200; i++ {
		{{$name}} := &{{.Type}}{}
		s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		{{$name}}s = append({{$name}}s, {{$name}})
		{{- if not $rocksDB}}
		s.NoError(legacyStore.Upsert(s.ctx, {{$name}}))
		{{- end}}
	}

    {{- if $rocksDB}}
    s.NoError(legacyStore.UpsertMany(s.ctx, {{$name}}s))
	{{- end}}

	// Move
	s.NoError(move(s.gormDB, s.pool, legacyStore))

	// Verify
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
