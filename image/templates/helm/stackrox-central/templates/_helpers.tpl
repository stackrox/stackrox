{{/*
    Misceallaneous helper templates.
   */}}

{{/*
  srox.expand $ $spec

  Parses and expands a "specification string" in the following way:
  - If $spec is a dictionary, return $spec rendered as a YAML.
  - Otherwise, if $spec starts with a backslash character (`\`), return $spec minus the leading
    backslash character.
  - Otherwise, if $spec starts with an `@` character, strip off the first character and
    treat the remainder of the string as a `|`-separated list of file names. Try to load
    each referenced file, in order, via `stackrox.getFile`. The result is the first file
    that could be successfully loaded. If no file could be loaded, expansion fails.
  - Otherwise, return $spec as-is.
   */}}
{{- define "srox.expand" -}}
{{- $ := index . 0 -}}
{{- $spec := index . 1 -}}
{{- $result := "" -}}
{{- if kindIs "string" $spec -}}
  {{- if hasPrefix "\\" $spec -}}
    {{- /* use \ as string-wide escape character */ -}}
    {{- $result = trimPrefix "\\" $spec -}}
  {{- else if hasPrefix "@" $spec -}}
    {{- /* treat as file list (first found matches) */ -}}
    {{- $fileList := regexSplit "\\s*\\|\\s*" ($spec | trimPrefix "@" | trim) -1 -}}
    {{- $fileRes := dict -}}
    {{- $_ := include "srox.loadFile" (list $ $fileRes $fileList) -}}
    {{- if not $fileRes.found -}}
      {{- include "srox.fail" (printf "Expanding reference %q: none of the referenced files were found" $spec) -}}
    {{- end -}}
    {{- $result = $fileRes.contents -}}
  {{- else -}}
    {{/* treat as raw string */}}
    {{- $result = $spec -}}
  {{- end -}}
{{- else if not (kindIs "invalid" $spec) -}}
  {{- /* render non-string, non-nil values as YAML */ -}}
  {{- $result = toYaml $spec -}}
{{- end -}}
{{- $result -}}
{{- end -}}


{{/*
  srox.loadFile $ $out $fileName-or-list

  This helper function reads a file. It differs from $.Files.Get in that it also takes
  $.Values.meta.fileOverrides into account. Furthermore, it can receive a list of file names,
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
    {{ $contents = index $.Values.meta.fileOverrides $fileName }}
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
