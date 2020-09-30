{{/*
  srox.expandAll $ $target $expandable [$path]

  Expands values within $target that are flagged in $expandable, using $path
  as the path from the configuration root to $target for error reporting purposes.

  If $target is nil, nothing happens. Otherwise, $target must be a dict. For every key
  of $target that is also present in $expandable, the following action is performed:
  - If the entry in $expandable is a dict, recursive invoke "srox.expandAll" on the
    respective entries, with an adjusted $path.
  - Otherwise, the entry in $expandable is assume to be of boolean value. If the value is
    true, the corresponding entry's value in $target is expanded (see "srox._expandSingle"
    below for a definition of expanding), and the result of the expansion is stored under
    the key with a "_" prepended in $target. The original entry in $target is removed. This
    ensures "srox.expandAll" is an idempotent operation).
   */}}
{{ define "srox.expandAll" }}
{{ $args := . }}
{{ $ := index $args 0 }}
{{ $target := index $args 1 }}
{{ $expandable := index $args 2 }}
{{ $path := list }}
{{ if ge (len $args) 4 }}
  {{ $path = index $args 3 }}
  {{ if kindIs "string" $path }}
    {{ $path = splitList "." $path | compact }}
  {{ end }}
{{ end }}

{{ if kindIs "map" $target }}
  {{ range $k, $v := $expandable }}
    {{ $childPath := append $path $k }}
    {{ $targetV := index $target $k }}
    {{ if kindIs "map" $v }}
      {{ include "srox.expandAll" (list $ $targetV $v $childPath) }}
    {{ else if $v }}
      {{ if not (kindIs "invalid" $targetV) }}
        {{ $expanded := include "srox._expandSingle" (list $ $targetV (join "." $childPath)) }}
        {{ $_ := set $target (printf "_%s" $k) $expanded }}
      {{ end }}
      {{ $_ := unset $target $k }}
    {{ end }}
  {{ end }}
{{ else if not (kindIs "invalid" $target) }}
  {{ include "srox.fail" (printf "Error expanding value at %s: expected map, got: %s" (join "." $path) (kindOf $target)) }}
{{ end }}
{{ end }}

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
{{- define "srox._expandSingle" -}}
    {{- $ := index . 0 -}}
    {{- $spec := index . 1 -}}
    {{- $context := index . 2 -}}
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
                {{- include "srox.fail" (printf "Expanding %s: file reference %q: none of the referenced files were found" $context $spec) -}}
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
