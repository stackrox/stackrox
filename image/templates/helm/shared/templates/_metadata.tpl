{{/*
  srox.labels $ $objType $objName [ $labels ]

  Format labels for $objType/$objName as YAML.
   */}}
{{- define "srox.labels" -}}
  {{- $ := index . 0 -}}
  {{- $objType := index . 1 -}}
  {{- $objName := index . 2 -}}
  {{- $labels := dict -}}
  {{- if gt (len .) 3 -}}
    {{- $labels = default dict (index . 3) -}}
  {{- end -}}
  {{- $resultingLabels := dict -}}
  {{- $_ := include "srox._labels" (list $ $resultingLabels $labels $objType $objName false) }}
  {{- toYaml $resultingLabels -}}
{{- end -}}

{{/*
  srox.podLabels $ $objType $objName [ $labels ]

  Format pod labels for $objType/$objName as YAML.
   */}}
{{- define "srox.podLabels" -}}
  {{- $ := index . 0 -}}
  {{- $objType := index . 1 -}}
  {{- $objName := index . 2 -}}
  {{- $labels := dict -}}
  {{- if gt (len .) 3 -}}
    {{- $labels = default dict (index . 3) -}}
  {{- end -}}
  {{- $resultingLabels := dict -}}
  {{- $_ := include "srox._labels" (list $ $resultingLabels $labels $objType $objName true) }}
  {{- toYaml $resultingLabels -}}
{{- end -}}

{{/*
  srox.annotations $ $objType $objName [ $annotations ]

  Format annotations for $objType/$objName as YAML.
   */}}
{{- define "srox.annotations" -}}
  {{- $ := index . 0 -}}
  {{- $objType := index . 1 -}}
  {{- $objName := index . 2 -}}
  {{- $annotations := dict -}}
  {{- if gt (len .) 3 -}}
    {{- $annotations = default dict (index . 3) -}}
  {{- end -}}
  {{- $resultingAnnotations := dict -}}
  {{- $_ := include "srox._annotations" (list $ $resultingAnnotations $annotations $objType $objName false) -}}
  {{- toYaml $resultingAnnotations -}}
{{- end -}}

{{/*
  srox.podAnnotations $ $objType $objName [ $annotations ]

  Format pod annotations for $objType/$objName as YAML.
   */}}
{{- define "srox.podAnnotations" -}}
  {{- $ := index . 0 -}}
  {{- $objType := index . 1 -}}
  {{- $objName := index . 2 -}}
  {{- $annotations := dict -}}
  {{- if gt (len .) 3 -}}
    {{- $annotations = default dict (index . 3) -}}
  {{- end -}}
  {{- $resultingAnnotations := dict -}}
  {{- $_ := include "srox._annotations" (list $ $resultingAnnotations $annotations $objType $objName true) -}}
  {{- toYaml $resultingAnnotations -}}
{{- end -}}

{{/*
  srox.envVars $ $objType $objName $containerName

  Format environment variables for container $containerName in
  $objType/$objName as YAML.
   */}}
{{- define "srox.envVars" -}}
{{- $envVars := dict -}}
{{- $_ := include "srox._envVars" (prepend . $envVars) -}}
{{- range $k := keys $envVars | sortAlpha -}}
{{- $v := index $envVars $k }}
- name: {{ quote $k }}
{{- if kindIs "map" $v }}
  {{- toYaml $v | nindent 2 }}
{{- else }}
  value: {{ quote $v }}
{{- end }}
{{ end -}}
{{- end -}}

{{/*
  srox._annotations $ $resultingAnnotations $annotations $objType $objName $forPod

  Writes all applicable [pod] annotations (including default annotations) for
  $objType/$objName into $annotations. Pod labels are written iff $forPod is true.

  This template receives the $ parameter as its second (not its first, as usual) parameter
  such that it can be used easier in "srox.annotations".
   */}}
{{ define "srox._annotations" }}
{{ $ := index . 0  }}
{{ $resultingAnnotations := index . 1 }}
{{ $annotations := index . 2 }}
{{ $objType := index . 3 }}
{{ $objName := index . 4 }}
{{ $forPod := index . 5 }}
{{ $_ := set $resultingAnnotations "meta.helm.sh/release-namespace" $.Release.Namespace }}
{{ $_ = set $resultingAnnotations "meta.helm.sh/release-name" $.Release.Name }}
{{ $_ = set $resultingAnnotations "owner" "stackrox" }}
{{ $_ = set $resultingAnnotations "email" "support@stackrox.com" }}
{{ $metadataNames := list "annotations" }}
{{ if $forPod }}
  {{ $metadataNames = append $metadataNames "podAnnotations" }}
{{ end }}
{{ include "srox._customizeMetadata" (list $ $resultingAnnotations $annotations $objType $objName $metadataNames) }}
{{ end }}

{{/*
  srox._envVars $envVars $ $objType $objName $containerName

  Writes all applicable environment variables for $objType/$objName
  into $envVars.

  This template receives the $ parameter as its second (not its first, as usual) parameter
  such that it can be used easier in "srox.envVars".
   */}}
{{ define "srox._envVars" }}
{{ $envVars := index . 0 }}
{{ $ := index . 1  }}
{{ $objType := index . 2 }}
{{ $objName := index . 3 }}
{{ $containerName := index . 4 }}
{{ $metadataNames := list "envVars" }}
{{ include "srox._customizeMetadata" (list $ $envVars dict $objType $objName $metadataNames) }}
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
  srox._customizeMetadata $ $resultingMetadata $metadata $objType $objName $metadataNames

  Writes custom key/value metadata to $metadata by consulting all sub-dicts with names in
  $metadataNames under the applicable custom metadata locations (._rox.customize,
  ._rox.customize.other.$objType/*, ._rox.customize.other.$objType/$objName, and
  ._rox.customizer.$objName [workloads only]). Dictionaries are consulted in this order, with
  values from dictionaries consulted later overwriting values from dictionaries consulted
  earlier.
   */}}
{{ define "srox._customizeMetadata" }}
{{ $ := index . 0 }}
{{ $resultingMetadata := index . 1 }}
{{ $metadata := index . 2 }}
{{ $objType := index . 3 }}
{{ $objName := index . 4 }}
{{ $metadataNames := index . 5 }}

{{ $overrideDictPaths := list "" (printf "other.%s/*" $objType) (printf "other.%s/%s" $objType $objName) }}
{{ if has $objType (list "deployment" "daemonset")  }}
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
      {{ include "srox.destructiveMergeOverwrite" (list $resultingMetadata $metadata $customMetadata) }}
    {{ end }}
  {{ end }}
{{ end }}
{{ end }}

{{/* Add namespace specific prefixes for global resources to avoid resource name clashes for multi-namespace deployments. */}}
{{- define "srox.globalResourceName" -}}
{{- $ := index . 0 -}}
{{- $name := index . 1 -}}

{{- if eq $.Release.Namespace "stackrox" -}}
  {{- /* Standard namespace, use resource name as is. */ -}}
  {{- $name -}}
{{- else -}}
  {{- /* Add global prefix to resource name. */ -}}
  {{- if hasPrefix "stackrox-" $name -}}
    {{- printf "%s-%s" $._rox.globalPrefix (trimPrefix "stackrox-" $name) -}}
  {{- else if hasPrefix "stackrox:" $name -}}
    {{- printf "%s:%s" $._rox.globalPrefix (trimPrefix "stackrox:" $name) -}}
  {{- else -}}
    {{- include "srox.fail" (printf "Unknown naming convention for global resource %q." $name) -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
    srox.initGlobalPrefix $

    Initializes prefix for global resources.
   */}}
{{- define "srox.initGlobalPrefix" -}}
{{- $ := index . 0 -}}
{{ if kindIs "invalid" $._rox.globalPrefix }}
  {{ if eq $.Release.Namespace "stackrox" }}
    {{ $_ := set $._rox "globalPrefix" "stackrox" }}
  {{ else }}
    {{ $_ := set $._rox "globalPrefix" (printf "stackrox-%s" (trimPrefix "stackrox-" $.Release.Namespace)) }}
  {{ end }}
{{ end }}

{{ if ne $._rox.globalPrefix "stackrox" }}
  {{ include "srox.note" (list $ (printf "Global Kubernetes resources are prefixed with '%s'." $._rox.globalPrefix)) }}
{{- end -}}
{{- end -}}

{{/*
  srox.getAnnotationTemplate . $name $out

  Retrieve the annotation template with the given $name and store it in the provided $out parameter.
   */}}
{{ define "srox.getAnnotationTemplate" }}
  {{ $ := index . 0 }}
  {{ $name := index . 1 }}
  {{ $out := index . 2 }}
  {{ if kindIs "invalid" $._rox._annotationTemplates }}
    {{ include "srox.fail" "Annotation templates not initialized" }}
  {{ end }}
  {{ $annotationTemplates := get $._rox._annotationTemplates $name }}
  {{ if not $annotationTemplates }}
    {{ include "srox.fail" (printf "Failed to retrieve annotation template %q" $name) }}
  {{ end }}
  {{ range $key, $value := $annotationTemplates }}
    {{ $_ := set $out $key $value }}
  {{ end }}
{{ end }}

{{/*
  srox.loadAnnotationTemplates .

  Load the annotation templates from `internal/annotations` and store them within $._rox.
  The templates can later be retrieved with `srox.getAnnotationTemplate`.
   */}}
{{ define "srox.initAnnotationTemplates" }}
  {{ $ := . }}
  {{ if kindIs "invalid" $._rox._annotationTemplates }}
    {{ $_ := set $._rox "_annotationTemplates" dict }}
  {{ end }}
  {{ range $fileName, $annotations := $.Files.Glob "internal/annotations/*.yaml" }}
    {{ $name := trimSuffix ".yaml" (base $fileName) }}
    {{ $_ := set $._rox._annotationTemplates $name ($annotations | toString | fromYaml) }}
  {{ end }}
{{ end }}
