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

{{/*
  srox.destructiveMergeOverwrite $out $dict1 $dict2...

  Recursively merges $dict1, $dict2 (in this order) into $out, similar to mergeOverwrite.
  The eponymous difference is the fact that any explicit "null" entries in the source
  dictionaries cause the respective entry to be deleted.
   */}}
{{ define "srox.destructiveMergeOverwrite" }}
{{ $out := first . }}
{{ $toMergeList := rest . }}
{{ range $toMerge := $toMergeList }}
  {{ range $k, $v := $toMerge }}
    {{ if kindIs "invalid" $v }}
      {{ $_ := unset $out $k }}
    {{ else if kindIs "map" $v }}
      {{ $outV := index $out $k }}
      {{ if kindIs "invalid" $outV }}
        {{ $_ := set $out $k (deepCopy $v) }}
      {{ else if kindIs "map" $outV }}
        {{ include "srox.destructiveMergeOverwrite" (list $outV $v) }}
      {{ else }}
        {{ fail (printf "when merging at key %s: incompatible kinds %s and %s" $k (kindOf $v) (kindOf $outV)) }}
      {{ end }}
    {{ else }}
      {{ $_ := set $out $k $v }}
    {{ end }}
  {{ end }}
{{ end }}
{{ end }}

{{/*
  srox.stringifyDictValues $dict

  Recursively traverses $dict and converts every non-dict value to a string.
   */}}
{{ define "srox.stringifyDictValues" }}
{{ $dict := . }}
{{ range $k, $v := $dict }}
  {{ if kindIs "map" $v }}
    {{ include "srox.stringifyDictValues" $v }}
  {{ else }}
    {{ $_ := set $dict $k (toString $v) }}
  {{ end }}
{{ end }}
{{ end }}

{{/*
  srox.safeDictLookup $dict $out $path

  Looks up $path in $dict, and stores the result (if any) in $out.result.
  $path is a dot-separated list of nested field names. An empty $path causes
  $dict to be stored in $out.result.

  Example: srox.safeDictLookup $dict $out "a.b.c" stores the value of $dict.a.b.c, if
  it exists, in $out.result. Otherwise, it does nothing - in particular, it does
  not fail, as accessing $dict.a.b.c unconditionally would if any of $dict, $dict.a,
  or $dict.a.b was not a dict.
   */}}
{{ define "srox.safeDictLookup" }}
{{ $dict := index . 0 }}
{{ $out := index . 1 }}
{{ $path := index . 2 }}
{{ $curr := $dict }}
{{ $pathList := splitList "." $path | compact }}
{{ range $pathElem := $pathList }}
  {{ if kindIs "map" $curr }}
    {{ $curr = index $curr $pathElem }}
  {{ else if not (kindIs "invalid" $curr) }}
    {{ $curr = dict.nil }}
  {{ end }}
{{ end }}
{{ if not (kindIs "invalid" $curr) }}
  {{ $_ := set $out "result" $curr }}
{{ end }}
{{ end }}



{{/*
  srox.mergeInto $tgt $src1..$srcN

  Recursively merges values from $src1, ..., $srcN into $tgt, giving preference to
  values in $tgt.

  Unlike Sprig's merge, this does not overwrite falsy values when explicitly defined,
  with the exception of `null` values (this also sets it apart from Sprig's mergeOverwrite).

  Whenever entire (nested) dicts are merged as-is from one of the sources into $tgt, a deep
  copy of the respective nested dict is created.

  An empty string is always returned, hence this should be invoked in the form
    $_ := include "srox.mergeInto" (list $tgt $src1 $src2)
   */}}
{{ define "srox.mergeInto" }}
{{ $tgt := first . }}
{{ range $src := rest . }}
  {{ range $k, $srcV := $src }}
    {{ $tgtV := index $tgt $k }}
    {{ if kindIs "map" $srcV }}
      {{ if kindIs "invalid" $tgtV }}
        {{ $_ := set $tgt $k (deepCopy $srcV) }}
      {{ else if kindIs "map" $tgtV }}
        {{ $_ := include "srox.mergeInto" (list $tgtV $srcV) }}
      {{ else }}
        {{ fail (printf "Incompatible kinds for key %s: %s vs %s" $k (kindOf $srcV) (kindOf $tgtV)) }}
      {{ end }}
    {{ else if and (not (kindIs "invalid" $srcV)) (kindIs "invalid" $tgtV) }}
      {{ $_ := set $tgt $k $srcV }}
    {{ end }}
  {{ end }}
{{ end }}
{{ end }}
