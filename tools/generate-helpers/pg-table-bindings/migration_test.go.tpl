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

	"github.com/gogo/protobuf/proto"
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
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
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

{{- if not .Migration.SingletonStore}}
func (s *postgresMigrationSuite) TestMigration() {
	// Prepare data and write to legacy DB
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
	s.NoError(move(s.gormDB, s.pool, legacyStore))
	var count int64
	s.gormDB.Model({{template  "createTableStmtVar" .Schema}}.GormModel).Count(&count)
	s.Equal(int64(len({{$name}}s)), count)
	for _, {{$name}} := range {{$name}}s {
		s.Equal({{$name}}, s.get({{ template "getterParamList" $ }}))
	}
	{{- /*
		sort.Slice({{$name}}s, func(i, j int) bool {
            return {{$name}}s[i].Id < {{$name}}s[j].Id
    })
	all, err := store.GetAll(s.ctx)
    s.Require().NoError(err)
    sort.Slice(all, func(i, j int) bool {
        return all[i].Id < all[j].Id
    })

    s.Equal({{$name}}s, all) */}}
}

func (s *postgresMigrationSuite) get({{template "paramList" $pks}}) *{{.Type}} {
{{/* TODO(ROX-10624): Remove this condition after all PKs fields were search tagged (PR #1653) */}}
{{- if eq (len $pks) 1 }}
    q := search.ConjunctionQuery(
    {{- range $idx, $pk := $pks}}
        {{- if eq $pk.Name $singlePK.Name }}
            search.NewQueryBuilder().AddDocIDs({{ $singlePK.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- else }}
            search.NewQueryBuilder().AddExactMatches(search.FieldLabel("{{ $pk.Search.FieldName }}"), {{ $pk.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- end}}
    {{- end}}
    )

	data, err := postgres.RunGetQueryForSchema(s.ctx, schema, q, s.pool)
	s.NoError(err)
{{- else }}
	conn, release, err := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
	s.NoError(err)
	defer release()

	row := conn.QueryRow(ctx, getStmt, {{template "argList" $pks}})
	var data []byte
	err = row.Scan(&data)
	s.NoError(pgutils.ErrNilIfNoRows(err))
{{- end }}
	var msg {{.Type}}
	s.NoError(proto.Unmarshal(data, &msg))
	return &msg
}
{{- else}}
func (s *postgresMigrationSuite) TestMigration() {
	// Prepare data and write to legacy DB
	legacyStore := legacy.New(s.legacyDB)
	store := pgStore.New(s.ctx, s.pool)
	{{$name}} := &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	s.NoError(legacyStore.Upsert(s.ctx, {{$name}}))
	s.NoError(move(s.gormDB, s.pool, legacyStore))
	fetched, found, err := store.Get(s.ctx)
	s.NoError(err)
	s.True(found)
	s.Equal({{$name}}, fetched)
}
{{- end}}
