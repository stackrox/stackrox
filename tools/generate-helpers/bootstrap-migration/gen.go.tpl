{{- define "TODO"}}TODO(do{{- /**/ -}}nt-merge){{- end}}

package postgres

//go:generate pg-schema-migration-helper --type={{.storeObject}} {{- if .tableProvided }}--table={{.tableName}} {{- end}} --schema-only --conversion-funcs --schema-directory --migration --package={{.packageName}} --migration-dir={{.migrationDir}} --data-access={{.dataAccess}}

// {{template "TODO"}}: remove this file as migrations should not be re-generated after being merged.
