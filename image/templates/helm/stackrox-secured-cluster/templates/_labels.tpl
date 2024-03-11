{{/*
  srox._labels $ $resultingLabels $labels $objType $objName $forPod

  Writes all applicable [pod] labels (including default labels) for $objType/$objName
  into $labels. Pod labels are written iff $forPod is true.

  This template receives the $ parameter as its second (not its first, as usual) parameter
  such that it can be used easier in "srox.labels".
   */}}
{{ define "srox._labels" }}
{{ $ := index . 0  }}
{{ $resultingLabels := index . 1 }}
{{ $labels := index . 2 }}
{{ $objType := index . 3 }}
{{ $objName := index . 4 }}
{{ $forPod := index . 5 }}
{{ $_ := set $resultingLabels "app.kubernetes.io/name" "stackrox" }}
{{ $_ = set $resultingLabels "app.kubernetes.io/managed-by" $.Release.Service }}
{{ $_ = set $resultingLabels "helm.sh/chart" (printf "%s-%s" $.Chart.Name ($.Chart.Version | replace "+" "_")) }}
{{ $_ = set $resultingLabels "app.kubernetes.io/instance" $.Release.Name }}
{{ $_ = set $resultingLabels "app.kubernetes.io/version" $.Chart.AppVersion }}
{{ $_ = set $resultingLabels "app.kubernetes.io/part-of" "stackrox-secured-cluster-services" }}
{{ $component := regexReplaceAll "^.*/(admission-control|collector|sensor)[^/]*\\.yaml" $.Template.Name "${1}" }}
{{ if not (contains "/" $component) }}
  {{ $_ = set $labels "app.kubernetes.io/component" $component }}
{{ end }}
{{ $metadataNames := list "labels" }}
{{ if $forPod }}
  {{ $metadataNames = append $metadataNames "podLabels" }}
{{ end }}
{{ include "srox._customizeMetadata" (list $ $resultingLabels $labels $objType $objName $metadataNames) }}
{{ end }}
