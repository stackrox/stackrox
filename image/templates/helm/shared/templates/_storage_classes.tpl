{{/*
  srox.getStorageClasses $

  This function attempts to retrieve information about all available StorageClasses on the
  cluster and write it to

    $._rox.env.storageClasses.all: A dict mapping storage class names to dicts containing
      relevant properties of the storage class.

    $._rox.env.storageClasses.default: Either nil or a string containing the name of the
      default StorageClass.
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

  {{ $_ := set $._rox.env.storageClasses "all" $storageClasses }}
  {{- if ne $defaultStorageClass "" -}}
    {{- $_ := set $._rox.env.storageClasses "default" $defaultStorageClass -}}
  {{- end -}}

{{- end -}}
