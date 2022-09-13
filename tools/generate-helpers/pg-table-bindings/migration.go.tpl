{{- define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}
{{- $ := . }}
{{- $name := .TrimmedType|lowerCamelCase }}
{{- $table := .Table|lowerCase }}
{{ $boltDB := eq .Migration.MigrateFromDB "boltdb" }}
{{ $dackbox := eq .Migration.MigrateFromDB "dackbox" }}
{{ $rocksDB := or $dackbox (eq .Migration.MigrateFromDB "rocksdb") }}

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	legacy "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/postgres"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
	{{if $dackbox}}rawDackbox "github.com/stackrox/rox/pkg/dackbox/raw"{{end}}
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres() + {{.Migration.MigrateSequence}},
		VersionAfter:   storage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres()) + {{add .Migration.MigrateSequence 1}}},
		Run: func(databases *types.Databases) error {
			{{- if $rocksDB}}
				{{- if $dackbox}}
				legacyStore := legacy.New(rawDackbox.GetGlobalDackBox(), rawDackbox.GetKeyFence())
				{{- else}}
				legacyStore, err := legacy.New(databases.PkgRocksDB)
				if err != nil {
					return err
				}
				{{- end}}
			{{- end}}
			{{- if $boltDB}}
			legacyStore := legacy.New(databases.BoltDB)
			{{- end}}
			if err := move(databases.GormDB, databases.PostgresDB, legacyStore); err != nil {
				return errors.Wrap(err,
					"moving {{.Table|lowerCase}} from rocksdb to postgres")
			}
			{{- if gt (len .Schema.RelationshipsToDefineAsForeignKeys) 0 }}
			if err := prune(databases.PostgresDB); err != nil {
				return errors.Wrap(err,
				"pruning {{.Table|lowerCase}}")
			}
			{{- end}}
			return nil
		},
	}
	batchSize	 = 10000
	schema		= {{template "schemaVar" .Schema}}
	log		   = loghelper.LogWrapper{}
)

{{$rocksDB :=  eq .Migration.MigrateFromDB "rocksdb" }}

func move(gormDB *gorm.DB, postgresDB *pgxpool.Pool, legacyStore legacy.Store) error {
	ctx := sac.WithAllAccess(context.Background())
	store := pgStore.New({{if .Migration.SingletonStore}}ctx, {{end}}postgresDB)
	pkgSchema.ApplySchemaForTable(context.Background(), gormDB, schema.Table)
	{{- if .Migration.SingletonStore}}
	obj, found, err := legacyStore.Get(ctx)
	if err != nil {
		log.WriteToStderr("failed to fetch {{$name}}")
		return err
	}
	if !found {
		return nil
	}
	if err = store.Upsert(ctx, obj); err != nil {
	log.WriteToStderrf("failed to persist object to store %v", err)
		return err
	}
	{{- else}}
	{{- /* Assume rocksdb and postgres agrees on if it should have GetAll function. Not acurate but works well. */}}
	{{- if or $rocksDB (not .GetAll) }}
	var {{.Table|lowerCamelCase}} []*{{.Type}}
	err := walk(ctx, legacyStore, func(obj *{{.Type}}) error {
		{{.Table|lowerCamelCase}} = append({{.Table|lowerCamelCase}}, obj)
		if len({{.Table|lowerCamelCase}}) == batchSize {
			if err := store.UpsertMany(ctx, {{.Table|lowerCamelCase}}); err != nil {
				log.WriteToStderrf("failed to persist {{.Table|lowerCase}} to store %v", err)
				return err
			}
			{{.Table|lowerCamelCase}} = {{.Table|lowerCamelCase}}[:0]
		}
		return nil
	})
	{{- else}}
	{{.Table|lowerCamelCase}}, err := legacyStore.GetAll(ctx)
	{{- end}}
	if err != nil {
		return err
	}
	if len({{.Table|lowerCamelCase}}) > 0 {
		if err = store.UpsertMany(ctx, {{.Table|lowerCamelCase}}); err != nil {
			log.WriteToStderrf("failed to persist {{.Table|lowerCase}} to store %v", err)
			return err
		}
	}
	{{- end}}
	return nil
}
{{if and (not .Migration.SingletonStore) (or $rocksDB (not .GetAll))}}
func walk(ctx context.Context, s legacy.Store, fn func(obj *{{.Type}}) error) error {
	{{- if $dackbox}}
	return store_walk(ctx, s, fn)
	{{- else}}
	return s.Walk(ctx, fn)
	{{- end}}
}
{{end}}

{{if $dackbox}}
func store_walk(ctx context.Context, s legacy.Store, fn func(obj *{{.Type}}) error) error {
	ids, err := s.GetIDs(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize

		if end > len(ids) {
			end = len(ids)
		}
		objs, _, err := s.GetMany(ctx, ids[i:end])
		if err != nil {
			return err
		}
		for _, obj := range objs {
			if err = fn(obj); err != nil {
				return err
			}
		}
	}
	return nil
}
{{end}}
{{- if gt (len (.Schema.RelationshipsToDefineAsForeignKeys)) 0 }}
func prune(postgresDB *pgxpool.Pool) error {
{{- range $idx, $rel := .Schema.RelationshipsToDefineAsForeignKeys }}
	ctx := sac.WithAllAccess(context.Background())
	deleteStmt := `DELETE FROM {{$table}} child WHERE NOT EXISTS
		(SELECT * FROM {{$rel.OtherSchema.Table}} parent WHERE
		{{range $idx2, $col := $rel.MappedColumnNames}}{{if $idx2}}AND {{end}}child.{{ $col.ColumnNameInThisSchema }} = parent.{{ $col.ColumnNameInOtherSchema }}{{end}})`
	log.WriteToStderr(deleteStmt)
	_, err := postgresDB.Exec(ctx, deleteStmt)
	if err != nil {
	log.WriteToStderrf("failed to clean up orphaned data for %s", schema.Table)
	return err
	}
	return nil
{{- end}}
}
{{- end}}

func init() {
	migrations.MustRegisterMigration(migration)
}
