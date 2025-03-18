{{/*
    srox.dumpVar $var

    This function pretty prints the value of the variable as JSON and fails evaluation.

    Use: {{- template "srox.dumpVar" $myVar }}

    Example: {{- template "srox.dumpVar" ._rox.admissionControl }}
*/}}
{{ define "srox.dumpVar" }}
{{- . | mustToPrettyJson | printf "\nThe JSON output of the dumped var is: \n%s" | fail }}
{{ end }}
