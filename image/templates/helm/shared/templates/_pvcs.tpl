{{/*
  srox.getPVCs $

  This function attempts to retrieve information about all available PVCs in the
  current namespace

    $._rox.env.pvcs.names: A list of PVC names.
   */}}
{{- define "srox.getPVCs" -}}

  {{- $ := index . 0 -}}
  {{- $pvcNames := list -}}
  {{- $lookupResult := dict -}}
  {{- $_ := include "srox.safeLookup" (list $ $lookupResult "v1" "PersistentVolumeClaim" $._rox._namespace "") -}}
  {{- range $pvc := get ($lookupResult.result | default dict) "items" | default list -}}
    {{- $pvcNames = append $pvcNames $pvc.metadata.name -}}
  {{- end -}}
  {{- $_ := set $._rox.env.pvcs "names" $pvcNames -}}

{{- end -}}
