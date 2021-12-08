{{/*
  srox._labels $labels $ $objType $objName $forPod

  Writes all applicable [pod] labels (including default labels) for $objType/$objName
  into $labels. Pod labels are written iff $forPod is true.

  This template receives the $ parameter as its second (not its first, as usual) parameter
  such that it can be used easier in "srox.labels".
   */}}
{{ define "srox._labels" }}
{{ $labels := index . 0 }}
{{ $ := index . 1  }}
{{ $objType := index . 2 }}
{{ $objName := index . 3 }}
{{ $forPod := index . 4 }}
{{ $_ := set $labels "app.kubernetes.io/name" "stackrox" }}
{{ $_ = set $labels "app.kubernetes.io/managed-by" $.Release.Service }}
{{ $_ = set $labels "helm.sh/chart" (printf "%s-%s" $.Chart.Name ($.Chart.Version | replace "+" "_")) }}
{{ $_ = set $labels "app.kubernetes.io/instance" $.Release.Name }}
{{ $_ = set $labels "app.kubernetes.io/version" $.Chart.AppVersion }}
{{ $_ = set $labels "app.kubernetes.io/part-of" "stackrox-central-services" }}
{{ $component := regexReplaceAll "^.*/\\d{2}-([a-z]+)-\\d{2}-[^/]+\\.yaml" $.Template.Name "${1}" }}
{{ if not (contains "/" $component) }}
  {{ $_ = set $labels "app.kubernetes.io/component" $component }}
{{ end }}
{{ $metadataNames := list "labels" }}
{{ if $forPod }}
  {{ $metadataNames = append $metadataNames "podLabels" }}
{{ end }}
{{ include "srox._customizeMetadata" (list $ $labels $objType $objName $metadataNames) }}
{{ end }}
