{{ $inMigration := ne (index . "Migration") nil}}
{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}}{{end}}{{end}}

{{- $ := . }}
{{- $pks := .Schema.PrimaryKeys }}

{{- $singlePK := false }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{/*If there are multiple pks, then use the explicitly specified id column.*/}}
{{- else if .Schema.ID.ColumnName}}
{{ $singlePK = .Schema.ID }}
{{- end }}
