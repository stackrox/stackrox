{{/*
  srox.getStorageClasses
   */}}
{{- define "srox.getStorageClasses" -}}

  {{- $ := index . 0 -}}
  {{- $storageClasses := dict -}}
  {{- $defaultStorageClass := "" -}}

  {{- $lookupResult := dict -}}
  {{- $_ := include "srox.safeLookup" (list $ $lookupResult "storage.k8s.io/v1" "StorageClass" "" "") -}}
  {{- range $sc := get ($lookupResult.result | default dict) "items" | default list -}}
    {{- $storageClassName := $sc.metadata.name -}}
    {{- $annotations := $sc.metadata.annotations | default dict -}}
    {{- $isDefault := index $annotations "storageclass.kubernetes.io/is-default-class" | default false -}}
    {{- $_ := set $storageClasses $storageClassName (dict "isDefault" $isDefault) -}}
    {{- if and $isDefault (not $defaultStorageClass) -}}
      {{- $defaultStorageClass = $storageClassName -}}
    {{- end -}}
  {{- end -}}

  {{ $_ := set $._rox.env.storageClasses "all" (dict "all" $storageClasses) }}

  {{- range $storageClassName, $storageClassProperties := $storageClasses -}}
    {{/* Would like to use `break`, but there are quite a few Helm 3.x versions out there not supporting that...
        On the other hand there should, of course, only be at most one default storage class on a cluster. ¯\_(ツ)_/¯ */}}
    {{- if and (get $storageClassProperties "isDefault") (not $defaultStorageClass) -}}
      {{- $defaultStorageClass = $storageClassName -}}
    {{- end -}}
  {{- end -}}

  {{- if ne $defaultStorageClass "" -}}
    {{- $_ := set $._rox.env.storageClasses "default" $defaultStorageClass -}}
  {{- end -}}

{{- end -}}
