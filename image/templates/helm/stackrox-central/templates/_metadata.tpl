{{/*
  srox.labels $ $objType $objName

  Format labels for $objType/$objName as YAML.
   */}}
{{- define "srox.labels" -}}
{{- $labels := dict -}}
{{- $_ := include "srox._labels" (append (prepend . $labels) false) -}}
{{- toYaml $labels -}}
{{- end -}}

{{/*
  srox.podLabels $ $objType $objName

  Format pod labels for $objType/$objName as YAML.
   */}}
{{- define "srox.podLabels" -}}
{{- $labels := dict -}}
{{- $_ := include "srox._labels" (append (prepend . $labels) true) -}}
{{- toYaml $labels -}}
{{- end -}}

{{/*
  srox.annotations $ $objType $objName

  Format annotations for $objType/$objName as YAML.
   */}}
{{- define "srox.annotations" -}}
{{- $annotations := dict -}}
{{- $_ := include "srox._annotations" (append (prepend . $annotations) false) -}}
{{- toYaml $annotations -}}
{{- end -}}

{{/*
  srox.podAnnotations $ $objType $objName

  Format pod annotations for $objType/$objName as YAML.
   */}}
{{- define "srox.podAnnotations" -}}
{{- $annotations := dict -}}
{{- $_ := include "srox._annotations" (append (prepend . $annotations) true) -}}
{{- toYaml $annotations -}}
{{- end -}}

{{/*
  srox.envVars $ $objType $objName $containerName

  Format environment variables for container $containerName in
  $objType/$objName as YAML.
   */}}
{{- define "srox.envVars" -}}
{{- $envVars := dict -}}
{{- $_ := include "srox._envVars" (prepend . $envVars) -}}
{{- range $k, $v := $envVars -}}
- name: {{ quote $k }}
  value: {{ quote $v }}
{{ end -}}
{{- end -}}

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
  {{ $_ = set $labels "app.kubernets.io/component" $component }}
{{ end }}
{{ $metadataNames := list "labels" }}
{{ if $forPod }}
  {{ $metadataNames = append $metadataNames "podLabels" }}
{{ end }}
{{ include "srox._customizeMetadata" (list $ $labels $objType $objName $metadataNames) }}
{{ end }}

{{/*
  srox._annotations $annotations $ $objType $objName $forPod

  Writes all applicable [pod] annotations (including default annotations) for
  $objType/$objName into $annotations. Pod labels are written iff $forPod is true.

  This template receives the $ parameter as its second (not its first, as usual) parameter
  such that it can be used easier in "srox.annotations".
   */}}
{{ define "srox._annotations" }}
{{ $annotations := index . 0 }}
{{ $ := index . 1  }}
{{ $objType := index . 2 }}
{{ $objName := index . 3 }}
{{ $forPod := index . 4 }}
{{ $_ := set $annotations "meta.helm.sh/release-namespace" $.Release.Namespace }}
{{ $_ = set $annotations "meta.helm.sh/release-name" $.Release.Name }}
{{ $_ = set $annotations "owner" "stackrox" }}
{{ $_ = set $annotations "email" "support@stackrox.com" }}
{{ $metadataNames := list "annotations" }}
{{ if $forPod }}
  {{ $metadataNames = append $metadataNames "podAnnotations" }}
{{ end }}
{{ include "srox._customizeMetadata" (list $ $annotations $objType $objName $metadataNames) }}
{{ end }}

{{/*
  srox._envVars $envVars $ $objType $objName $containerName

  Writes all applicable [pod] labels (including default labels) for $objType/$objName
  into $labels. Pod labels are written iff $forPod is true.

  This template receives the $ parameter as its second (not its first, as usual) parameter
  such that it can be used easier in "srox.labels".
   */}}
{{ define "srox._envVars" }}
{{ $envVars := index . 0 }}
{{ $ := index . 1  }}
{{ $objType := index . 2 }}
{{ $objName := index . 3 }}
{{ $containerName := index . 4 }}
{{ $metadataNames := list "envVars" }}
{{ include "srox._customizeMetadata" (list $ $envVars $objType $objName $metadataNames) }}
{{ if $containerName }}
  {{ $containerKey := printf "/%s" $containerName }}
  {{ $envVarsForContainer := index $envVars $containerKey }}
  {{ if $envVarsForContainer }}
    {{ include "srox.destructiveMergeOverwrite" (list $envVars $envVarsForContainer) }}
  {{ end }}
{{ end }}

{{/* Remove all entries starting with / */}}
{{ range $key, $_ := $envVars }}
  {{ if hasPrefix "/" $key }}
    {{ $_ := unset $envVars $key }}
  {{ end }}
{{ end }}
{{ end }}

{{/*
  srox._customizeMetadata $ $metadata $objType $objName $metadataNames

  Writes custom key/value metadata to $metadata by consulting all sub-dicts with names in
  $metadataNames under the applicable custom metadata locations (._rox.customize,
  ._rox.customize.other.$objType/*, ._rox.customize.other.$objType/$objName, and
  ._rox.customizer.$objName [workloads only]). Dictionaries are consulted in this order, with
  values from dictionaries consulted later overwriting values from dictionaries consulted
  earlier.
   */}}
{{ define "srox._customizeMetadata" }}
{{ $ := index . 0 }}
{{ $metadata := index . 1 }}
{{ $objType := index . 2 }}
{{ $objName := index . 3 }}
{{ $metadataNames := index . 4 }}

{{ $overrideDictPaths := list "" (printf "other.%s/*" $objType) (printf "other.%s/%s" $objType $objName) }}
{{ if eq $objType "deployment" }}
  {{ $overrideDictPaths = append $overrideDictPaths $objName }}
{{ end }}

{{ range $dictPath := $overrideDictPaths }}
  {{ $customizeDict := $._rox.customize }}
  {{ if $dictPath }}
    {{ $resolvedOut := dict }}
    {{ include "srox.safeDictLookup" (list $._rox.customize $resolvedOut $dictPath) }}
    {{ $customizeDict = $resolvedOut.result }}
  {{ end }}
  {{ if $customizeDict }}
    {{ range $metadataName := $metadataNames }}
      {{ $customMetadata := index $customizeDict $metadataName }}
      {{ include "srox.destructiveMergeOverwrite" (list $metadata $customMetadata) }}
    {{ end }}
  {{ end }}
{{ end }}
{{ end }}
