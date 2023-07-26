{{/*
    srox.safeLookup $ $out $apiVersion $kind $ns $name

    This function does nothing if $.meta.useLookup is false; otherwise, it will
    perform a `lookup $apiVersion $kind $ns $name` operation and store the result in
    $out.result.

    Additionally, if a lookup was attempted, $out.reliable will contain a bool indicating
    whether the result of lookup can be relied upon. This is determined to be the case if
    the default service account in the release namespace can be found.
   */}}
{{ define "srox.safeLookup" }}
{{ $ := index . 0 }}
{{ $out := index . 1 }}
{{ if $._rox.meta.useLookup }}
  {{ if kindIs "invalid" $._rox._state.lookupWorks }}
    {{ $testOut := dict }}
    {{ include "srox._doLookup" (list $ $testOut "v1" "ServiceAccount" $.Release.Namespace "default") }}
    {{ $_ := set $._rox._state "lookupWorks" ($testOut.result | not | not) }}
  {{ end }}
  {{ include "srox._doLookup" . }}
  {{ $_ := set $out "reliable" $._rox._state.lookupWorks }}
{{ end }}
{{ end }}


{{/*
    srox._doLookup $ $out $apiVersion $kind $ns $name

    Calls "lookup" with arguments $apiVersion $kind $ns $name, and stores the result
    in $out.result.

    This function exists to prevent a parse error if the lookup function isn't defined. It does
    so by deferring the execution of lookup to a template string instantiated via `tpl`.
   */}}
{{ define "srox._doLookup" }}
{{ $ := index . 0 }}
{{ $tplArgs := dict "Template" $.Template "out" (index . 1) "apiVersion" (index . 2) "kind" (index . 3) "ns" (index . 4) "name" (index . 5) }}
{{ $_ := tpl "{{ $_ := set .out \"result\" (lookup .apiVersion .kind .ns .name) }}" $tplArgs }}
{{ end }}
