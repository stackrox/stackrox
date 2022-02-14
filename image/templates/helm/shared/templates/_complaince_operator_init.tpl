{{ define "compliance.init" }}
/*
Receive values configuration
*/
{{ $ := index . 1 }}

{{ include "compliance.Deployment" $ }}
{{ include "compliance.Secrets" $ }}
{{ end }}
