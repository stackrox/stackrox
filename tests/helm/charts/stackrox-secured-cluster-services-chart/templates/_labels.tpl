{{/*
  srox._labels $ $labels $extraLabels $objType $objName $forPod

  Writes all applicable [pod] labels (including default labels) for $objType/$objName
  into $labels. Pod labels are written iff $forPod is true.
  The dict $extraLabels can be used for specifying additional labels which
  can be modified using `customize` entries before before they are added to $labels.
   */}}
{{ define "srox._labels" }}
{{ $ := index . 0  }}
{{ $labels := index . 1 }}
{{ $extraLabels := index . 2 }}
{{ $objType := index . 3 }}
{{ $objName := index . 4 }}
{{ $forPod := index . 5 }}
{{ $_ := set $labels "app.kubernetes.io/name" "stackrox" }}
{{ $_ = set $labels "app.kubernetes.io/managed-by" $.Release.Service }}
{{ $_ = set $labels "helm.sh/chart" (printf "%s-%s" $.Chart.Name ($.Chart.Version | replace "+" "_")) }}
{{ $_ = set $labels "app.kubernetes.io/instance" $.Release.Name }}
{{ $_ = set $labels "app.kubernetes.io/version" $.Chart.AppVersion }}
{{ $_ = set $labels "app.kubernetes.io/part-of" "stackrox-secured-cluster-services" }}
{{ $component := regexReplaceAll "^.*/(\\d{2}-)?(admission-control|collector|sensor|scanner-v4)[^/]*\\.yaml" $.Template.Name "${2}" }}
{{ if not (contains "/" $component) }}
  {{ $_ = set $labels "app.kubernetes.io/component" $component }}
{{ end }}
{{ $metadataNames := list "labels" }}
{{ if $forPod }}
  {{ $metadataNames = append $metadataNames "podLabels" }}
{{ end }}
{{ include "srox._customizeMetadata" (list $ $labels $extraLabels $objType $objName $metadataNames) }}
{{ end }}
