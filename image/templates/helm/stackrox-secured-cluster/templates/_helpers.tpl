{{/*
    Misceallaneous helper templates.
   */}}




{{/*
  srox.loadFile $ $out $fileName-or-list

  This helper function reads a file. It differs from $.Files.Get in that it also takes
  $._rox.meta.fileOverrides into account. Furthermore, it can receive a list of file names,
  and will try these files in order. Finally, it indicates whether a file was found via the
  $out.found property (as opposed to $.Files.Get, which cannot distinguish between a successful
  read of an empty file, and this file not being found).
  The file contents will be returned via $out.contents
   */}}
{{ define "srox.loadFile" }}
{{ $ := index . 0 }}
{{ $out := index . 1 }}
{{ $fileNames := index . 2 }}
{{ if not (kindIs "slice" $fileNames) }}
  {{ $fileNames = list $fileNames }}
{{ end }}
{{ $contents := index dict "" }}
{{ range $fileName := $fileNames }}
  {{ if kindIs "invalid" $contents }}
    {{ $contents = index $._rox.meta.fileOverrides $fileName }}
  {{ end }}
  {{ if kindIs "invalid" $contents }}
    {{ range $path, $_ := $.Files.Glob $fileName }}
      {{ if kindIs "invalid" $contents }}
        {{ $contents = $.Files.Get $path }}
      {{ end }}
    {{ end }}
  {{ end }}
{{ end }}
{{ if not (kindIs "invalid" $contents) }}
  {{ $_ := set $out "contents" $contents }}
{{ end }}
{{ $_ := set $out "found" (not (kindIs "invalid" $contents)) }}
{{ end }}


{{/*
  srox.checkGenerated $ $cfgPath

  Checks if the value at configuration path $cfgPath (e.g., "central.adminPassword.value") was
  generated. Evaluates to the string "true" if this is the case, and an empty string otherwise.
   */}}
{{- define "srox.checkGenerated" -}}
{{- $ := index . 0 -}}
{{- $cfgPath := index . 1 -}}
{{- $genCfg := $._rox._state.generated -}}
{{- $exists := true -}}
{{- range $pathElem := splitList "." $cfgPath -}}
  {{- if $exists -}}
    {{- if hasKey $genCfg $pathElem -}}
      {{- $genCfg = index $genCfg $pathElem -}}
    {{- else -}}
      {{- $exists = false -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if $exists -}}
true
{{- end -}}
{{- end -}}
