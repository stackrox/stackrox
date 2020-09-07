{{/*
  srox.compactDict $target [$depth]

  Compacts a dict $target by removing entries with empty values.
  By default, only the top-level dict $target itself is modified. If the optional $depth
  parameter is specified and is non-zero, this determines the recursion depth over which the
  compaction is applied to nested diocts as well. A $depth of -1 means to compact all nested
  dicts, regardless of depth.
   */}}
{{ define "srox.compactDict" }}
{{ $args := . }}
{{ if not (kindIs "slice" $args) }}
  {{ $args = list $args 0 }}
{{ end }}
{{ $target := index $args 0 }}
{{ $depth := index $args 1 }}
{{ $zeroValKeys := list }}
{{ range $k, $v := $target }}
  {{ if and (kindIs "map" $v) (ne $depth 0) }}
    {{ include "srox.compactDict" (list $v (sub $depth 1)) }}
  {{ end }}
  {{ if not $v }}
    {{ $zeroValKeys = append $zeroValKeys $k }}
  {{ end }}
{{ end }}
{{ range $k := $zeroValKeys }}
  {{ $_ := unset $target $k }}
{{ end }}
{{ end }}
