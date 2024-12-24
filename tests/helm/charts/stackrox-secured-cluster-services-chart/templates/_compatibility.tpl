{{ define "srox.applyCompatibilityTranslation" }}
{{ $ := index . 0 }}
{{ $values := index . 1 }}
{{ $translationRules := $.Files.Get "internal/compatibility-translation.yaml" | fromYaml }}
{{ include "srox._doApplyCompat" (list $values $.Template $values $translationRules list) }}
{{ end }}

{{ define "srox._doApplyCompat" }}
{{ $values := index . 0 }}
{{ $template := index . 1 }}
{{ $valuesCtx := index . 2 }}
{{ $ruleCtx := index . 3 }}
{{ $ctxPath := index . 4 }}
{{ range $k, $v := $ruleCtx }}
  {{ $oldVal := index $valuesCtx $k }}
  {{ if not (kindIs "invalid" $oldVal) }}
    {{ if kindIs "map" $v }}
      {{ if kindIs "map" $oldVal }}
        {{ include "srox._doApplyCompat" (list $values $template $oldVal $v (append $ctxPath $k)) }}
        {{ if not $oldVal }}
          {{ $_ := unset $valuesCtx $k }}
        {{ end }}
      {{ end }}
    {{ else }}
      {{ $_ := unset $valuesCtx $k }}
      {{ if not (kindIs "invalid" $v) }}
        {{ $tplCtx := dict "Template" $template "value" (toJson $oldVal) "rawValue" $oldVal }}
        {{ $configFragment := tpl $v $tplCtx | fromYaml }}
        {{ include "srox._mergeCompat" (list $values $configFragment (append $ctxPath $k) list) }}
      {{ end }}
    {{ end }}
  {{ end }}
{{ end }}
{{ end }}

{{ define "srox._mergeCompat" }}
{{ $values := index . 0 }}
{{ $newConfig := index . 1 }}
{{ $compatValuePath := index . 2 }}
{{ $path := index . 3 }}
{{ range $k, $v := $newConfig }}
  {{ $currVal := index $values $k }}
  {{ if kindIs "invalid" $currVal }}
    {{ $_ := set $values $k $v }}
  {{ else if and (kindIs "map" $v) (kindIs "map" $currVal) }}
    {{ include "srox._mergeCompat" (list $currVal $v $compatValuePath (append $path $k)) }}
  {{ else }}
    {{ include "srox.fail" (printf "Conflict between legacy configuration values %s and explicitly set configuration value %s, please unset legacy value" (join "." $compatValuePath) (append $path $k | join ".")) }}
  {{ end }}
{{ end }}
{{ end }}
