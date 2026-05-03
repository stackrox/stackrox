{{/*
  srox.applyDefaults .

  Applies defaults defined in `internal/defaults`, in an order that depends on the filenames.
   */}}
{{ define "srox.applyDefaults" }}
{{ $ := . }}
{{/* Apply defaults */}}
{{ range $defaultsFile, $defaultsTpl := $.Files.Glob "internal/defaults/*.yaml" }}
  {{ $tplSects := regexSplit "(^|\n)---($|\n)" (toString $defaultsTpl) -1 }}
  {{ $sectCounter := 0 }}
  {{ range $tplSect := $tplSects }}
    {{/*
      tpl will merely stop creating output if an error is encountered during rendering (not during parsing), but we want
      to be certain that we recognized invalid templates. Hence, add a marker line at the end, and verify that it
      shows up in the output.
      */}}
    {{ $renderedSect := tpl (list $tplSect "{{ \"\\n#MARKER\\n\" }}" | join "") $ }}
    {{ if not (hasSuffix "\n#MARKER\n" $renderedSect) }}
      {{ include "srox.fail" (printf "Section %d in defaults file %s contains invalid templating" $sectCounter $defaultsFile) }}
    {{ end }}
    {{/*
      fromYaml only returns an empty dict upon error, but we want to be certain that we recognized invalid YAML.
      Hence, add a marker value.
      */}}
    {{ $sectDict := fromYaml (cat $renderedSect "\n__marker: true\n") }}
    {{ if not (index $sectDict "__marker") }}
      {{ include "srox.fail" (printf "Section %d in defaults file %s contains invalid YAML" $sectCounter $defaultsFile) }}
    {{ end }}
    {{ $_ := unset $sectDict "__marker" }}
    {{ $_ = include "srox.mergeInto" (list $._rox $sectDict) }}
    {{ $sectCounter = add $sectCounter 1 }}
  {{ end }}
{{ end }}
{{ end }}

{{/*
  srox.ensureCentralEndpointContainsPort .

  Appends a default port to the configured central endpoint based on a very simply heuristic.
  Specifically, it only checks if the provided endpoint contains a prefix "https://" and
  the part after that prefix does not contain a double colon.
  This heuristic is kept simple on purpose and does not correctly add the default ports in
  case the host part is an IPv6 address.
*/}}
{{ define "srox.ensureCentralEndpointContainsPort" }}
  {{ $ := . }}

  {{ $endpoint := $._rox.centralEndpoint }}
  {{ if hasPrefix "https://" $endpoint }}
    {{ $endpoint = trimPrefix "https://" $endpoint }}
    {{ if not (contains ":" $endpoint) }}
      {{ include "srox.note" (list $ (printf "Specified centralEndpoint %s does not contain a port, assuming port 443. If this is incorrect please specify the correct port." $._rox.centralEndpoint)) }}
      {{ $_ := set $._rox "centralEndpoint" (printf "%s:443" $._rox.centralEndpoint) }}
    {{ end }}
  {{ end }}

{{ end }}
