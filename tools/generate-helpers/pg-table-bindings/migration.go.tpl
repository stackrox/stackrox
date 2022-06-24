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
	legacy "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/{{.Migration.Dir}}/postgres"
	"github.com/stackrox/rox/migrator/types"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: 100,
		VersionAfter:   storage.Version{SeqNum: 101},
		Run: func(databases *types.Databases) error {
		    {{- if $rocksDB}}
		    legacyStore, err := legacy.New(databases.{{if $rocksDB}}PkgRocksDB{{else}}BoltDB{{end}})
		    if err != nil {
		        return err
		    }
		    {{- else}}
		    legacyStore := legacy.New(databases.{{if $rocksDB}}PkgRocksDB{{else}}BoltDB{{end}})
		    {{- end}}
			if err := move({{if $rocksDB}}databases.PkgRocksDB{{else}}databases.BoltDB{{- end}}, databases.GormDB, databases.PostgresDB, legacyStore); err != nil {
				return errors.Wrap(err,
					"moving {{.Table|lowerCase}} from rocksdb to postgres")
			}
			return nil
		},
	}
	batchSize     = 10000
	schema        = {{template "schemaVar" .Schema}}
	log           = loghelper.LogWrapper{}
)

{{$rocksDB :=  eq .Migration.MigrateFromDB "rocksdb" }}

func move(legacyDB {{if $rocksDB}}*rocksdb.RocksDB{{else}}*bolt.DB{{end}}, gormDB *gorm.DB, postgresDB *pgxpool.Pool, legacyStore legacy.Store) error {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	store := pgStore.New({{if .Migration.SingletonStore}}ctx, {{end}}postgresDB)
	pkgSchema.ApplySchemaForTable(context.Background(), gormDB, schema.Table)
	{{- if .Migration.SingletonStore}}
    	config, found, err := legacyStore.Get(ctx)
    	if err != nil {
            log.WriteToStderr("failed to fetch {{$name}}")
            return err
        }
        if !found {
            return nil
        }
        if err = store.Upsert(ctx, config); err != nil {
        log.WriteToStderrf("failed to persist configs to store %v", err)
            return err
        }
    {{- else}}
	    {{- if or $rocksDB (not .GetAll) }}
	    var {{.Table|lowerCamelCase}} []*{{.Type}}
	    var err error
	    legacyStore.Walk(ctx, func(obj *{{.Type}}) error {
		    {{.Table|lowerCamelCase}} = append({{.Table|lowerCamelCase}}, obj)
		    if len({{.Table|lowerCamelCase}}) == 10*batchSize {
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
        if err != nil {
            log.WriteToStderr("failed to fetch all {{.Table|lowerCamelCase}}")
            return err
        }
	    {{- end}}
	    if len({{.Table|lowerCamelCase}}) > 0 {
		    if err = store.UpsertMany(ctx, {{.Table|lowerCamelCase}}); err != nil {
			    log.WriteToStderrf("failed to persist {{.Table|lowerCase}} to store %v", err)
			    return err
		    }
	    }
    {{- end}}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
