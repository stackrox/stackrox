{{- define "TODO"}}TODO(do{{- /**/ -}}nt-merge){{- end}}

package store

//go:generate pg-schema-migration-helper --type={{.storeObject}} --table={{.tableName}} --schema-only --conversion-funcs --schema-directory --migration --package={{.packageName}} --migration-dir={{.migrationDir}}

// {{template "TODO"}}: remove this file as migrations should not be re-generated after being merged.
