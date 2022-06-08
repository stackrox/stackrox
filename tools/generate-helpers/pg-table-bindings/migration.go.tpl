{{- define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{- define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}
{{- $ := . }}
{{- $pks := .Schema.PrimaryKeys }}
{{- $singlePK := false }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{/*If there are multiple pks, then use the explicitly specified id column.*/}}
{{- else if .Schema.ID.ColumnName}}
{{ $singlePK = .Schema.ID }}
{{- end }}

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
    ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/db"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: 100,
		VersionAfter:   storage.Version{SeqNum: 101},
		Run: func(databases *types.Databases) error {
			if err := move{{.Table|upperCamelCase}}(databases.PkgRocksDB, databases.GormDB, databases.PostgresDB); err != nil {
				return errors.Wrap(err,
					"moving {{.Table|lowerCase}} from rocksdb to postgres")
			}
			return nil
		},
	}
	rocksdbBucket = []byte("{{.Migration.MigrateFromBucket}}")
	batchSize     = 10000
	schema        = {{template "schemaVar" .Schema}}
	log           = loghelper.LogWrapper{}
)

func move{{.Table|upperCamelCase}}(rocksDB *rocksdb.RocksDB, gormDB *gorm.DB, postgresDB *pgxpool.Pool) error {
	ctx := context.Background()
	store := newStore(postgresDB, generic.NewCRUD(rocksDB, rocksdbBucket, keyFunc, alloc, false))
	pkgSchema.ApplySchemaForTable(context.Background(), gormDB, schema.Table)

	var {{.Table|lowerCamelCase}} []*{{.Type}}
	store.Walk(ctx, func(obj *{{.Type}}) error {
		{{.Table|lowerCamelCase}} = append({{.Table|lowerCamelCase}}, obj)
		if len({{.Table|lowerCamelCase}}) == 10*batchSize {
			if err := store.copyFrom(ctx, {{.Table|lowerCamelCase}}...); err != nil {
				log.WriteToStderrf("failed to persist {{.Table|lowerCase}} to store %v", err)
				return err
			}
			{{.Table|lowerCamelCase}} = {{.Table|lowerCamelCase}}[:0]
		}
		return nil
	})
	if len({{.Table|lowerCamelCase}}) > 0 {
		if err := store.copyFrom(ctx, {{.Table|lowerCamelCase}}...); err != nil {
			log.WriteToStderrf("failed to persist {{.Table|lowerCase}} to store %v", err)
			return err
		}
	}
	return nil
}

type storeImpl struct {
	db   *pgxpool.Pool // Postgres DB
	crud db.Crud // Rocksdb DB crud
}

// newStore returns a new Store instance using the provided sql instance.
func newStore(db *pgxpool.Pool, crud db.Crud) *storeImpl {
	return &storeImpl{
		db:   db,
		crud: crud,
	}
}

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*pgxpool.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

{{- if not .JoinTable }}
func (s *storeImpl) DeleteMany(ctx context.Context, ids []{{$singlePK.Type}}) error {
	q := search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery()
	return postgres.RunDeleteRequestForSchema(schema, q, s.db)
}
{{end}}

func init() {
	migrations.MustRegisterMigration(migration)
}
