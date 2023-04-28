{{- define "schemaVar"}}frozenSchema.{{.Table|upperCamelCase}}Schema{{end}}
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}
{{- $ := . }}
{{- $name := .TrimmedType|lowerCamelCase }}
{{ $boltDB := eq .Migration.MigrateFromDB "boltdb" }}
{{ $dackbox := eq .Migration.MigrateFromDB "dackbox" }}
{{ $rocksDB := or $dackbox (eq .Migration.MigrateFromDB "rocksdb") }}

import (
	"context"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/pkg/errors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	legacy "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/postgres"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
	{{if $dackbox}}rawDackbox "github.com/stackrox/rox/pkg/dackbox/raw"{{end}}
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres() + {{.Migration.MigrateSequence}},
		VersionAfter:   &storage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres()) + {{add .Migration.MigrateSequence 1}}},
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
			return nil
		},
	}
	batchSize	 = {{.Migration.BatchSize}}
	schema		= {{template "schemaVar" .Schema}}
	log		   = loghelper.LogWrapper{}
)

{{$rocksDB :=  eq .Migration.MigrateFromDB "rocksdb" }}

func move(gormDB *gorm.DB, postgresDB postgres.DB, legacyStore legacy.Store) error {
	ctx := sac.WithAllAccess(context.Background())
	store := pgStore.New(postgresDB)
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTable{{.Table|upperCamelCase}}Stmt)

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
	identifiers, err := s.GetIDs(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < len(identifiers); i += batchSize {
		end := i + batchSize

		if end > len(identifiers) {
			end = len(identifiers)
		}
		objs, _, err := s.GetMany(ctx, identifiers[i:end])
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

func init() {
	migrations.MustRegisterMigration(migration)
}
