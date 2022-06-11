{{define "createTableStmtVar"}}pkgSchema.CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{- $name := .TrimmedType|lowerCamelCase }}
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}
{{define "getterParamList"}}{{$name := .TrimmedType|lowerCamelCase}}{{range $idx, $pk := .Schema.PrimaryKeys}}{{if $idx}}, {{end}}{{$pk.Getter $name}}{{end}}{{end}}

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	"gorm.io/gorm"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	ctx         context.Context

	// RocksDB
	rocksDB *rocksdb.RocksDB
	db      *gorocksdb.DB

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
	s.rocksDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = s.rocksDB.DB

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	s.ctx = context.Background()
	s.pool, err = pgxpool.ConnectConfig(s.ctx, config)
	s.Require().NoError(err)

	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	_ = s.gormDB.Migrator().DropTable({{template "createTableStmtVar" .Schema}}.GormModel)
}

func (s *postgresMigrationSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksDB)
	_ = s.gormDB.Migrator().DropTable({{template "createTableStmtVar" .Schema}}.GormModel)
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *postgresMigrationSuite) TestMigration() {
	// Prepare data and write to legacy DB
	batchSize = 48
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	var {{$name}}s []*{{.Type}}
	for i := 0; i < 200; i++ {
		{{$name}} := &{{.Type}}{}
		s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		bytes, err := proto.Marshal({{$name}})
		s.NoError(err, "failed to marshal data")
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(rocksdbBucket, keyFunc({{$name}})), bytes)
		{{$name}}s = append({{$name}}s, {{$name}})
	}

	s.NoError(s.db.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch))
	s.NoError(move{{.Table|upperCamelCase}}(s.rocksDB, s.gormDB, s.pool))
	var count int64
	s.gormDB.Model({{template  "createTableStmtVar" .Schema}}.GormModel).Count(&count)
	s.Equal(int64(len({{$name}}s)), count)
	for _, {{$name}} := range {{$name}}s {
		s.Equal({{$name}}, s.get({{ template "getterParamList" $ }}))
	}
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

