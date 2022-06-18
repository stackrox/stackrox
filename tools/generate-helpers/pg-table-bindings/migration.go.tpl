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
{{- $name := .TrimmedType|lowerCamelCase }}
{{ $rocksDB := eq .Migration.MigrateFromDB "rocksdb" }}

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
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: 100,
		VersionAfter:   storage.Version{SeqNum: 101},
		Run: func(databases *types.Databases) error {
			if err := move{{.Table|upperCamelCase}}({{if $rocksDB}}databases.PkgRocksDB{{else}}databases.BoltDB{{- end}}, databases.GormDB, databases.PostgresDB); err != nil {
				return errors.Wrap(err,
					"moving {{.Table|lowerCase}} from rocksdb to postgres")
			}
			return nil
		},
	}
	{{$name}}Bucket = []byte("{{.Migration.MigrateFromBucket}}")
	batchSize     = 10000
	schema        = {{template "schemaVar" .Schema}}
	log           = loghelper.LogWrapper{}
)

{{$rocksDB :=  eq .Migration.MigrateFromDB "rocksdb" }}
func move{{.Table|upperCamelCase}}(legacyDB {{if $rocksDB}}*rocksdb.RocksDB{{else}}*bolt.DB{{end}}, gormDB *gorm.DB, postgresDB *pgxpool.Pool) error {
	ctx := context.Background()
	store := newStore(postgresDB, {{if $rocksDB}}generic.NewCRUD(legacyDB, {{$name}}Bucket, keyFunc, alloc, false){{else}}legacyDB{{end}})
	pkgSchema.ApplySchemaForTable(context.Background(), gormDB, schema.Table)

	var {{.Table|lowerCamelCase}} []*{{.Type}}
	var err error
	{{- if $rocksDB}}
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
	{{- else}}
	{{.Table|lowerCamelCase}}, err = store.GetAll(ctx)
    if err != nil {
        log.WriteToStderr("failed to fetch all {{.Table|lowerCamelCase}}")
        return err
    }
	{{- end}}
	if len({{.Table|lowerCamelCase}}) > 0 {
		if err = store.copyFrom(ctx, {{.Table|lowerCamelCase}}...); err != nil {
			log.WriteToStderrf("failed to persist {{.Table|lowerCase}} to store %v", err)
			return err
		}
	}
	return nil
}

type storeImpl struct {
	db   *pgxpool.Pool // Postgres DB
	{{- if $rocksDB}}
	crud db.Crud // Rocksdb DB crud
	{{- else}}
	legacyDB *bolt.DB
	{{- end}}
}

// newStore returns a new Store instance using the provided sql instance.
func newStore(db *pgxpool.Pool, {{if $rocksDB}}crud db.Crud{{else}}legacyDB *bolt.DB{{end}}) *storeImpl {
	return &storeImpl{
		db:   db,
		{{- if $rocksDB}}crud: crud{{else}}legacyDB: legacyDB{{end}},
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
