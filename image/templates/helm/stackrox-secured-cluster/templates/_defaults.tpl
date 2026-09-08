{{/*
  srox.collectDefaults .

  Retrieves the defaults defined in `internal/defaults`, in an order that depends on the filenames.
   */}}
{{ define "srox.collectDefaults" }}
{{ $ := . }}
{{/*
  We don't populate $._rox directly here, instead we store the retrieved defaults in $._rox._defaults.
  But this undertaking is complicated by the fact that the files in internal/defaults are constructing the
  final hierarchy of default values iteratively, in particular they are building upon what has already been stored in
  the provided top-level context by a previous defaulting fragment.
  (Note that they do not only access "$._rox", but also non-stackrox fields, which are added to the top-level
  context by Helm, e.g. .Release.)

  Therefore, we are creating a temporary context $tplCtx for the templating of the default files here, which is the
  result of merging the "." context as passed to this macro and the defaults collected so far.

  After having invoked the templating we need to make sure to transfer updates to $tplCtx._rox._state back to the original
  context, otherwise invocations of "srox.warn" or "srox.note" during the templating would be lost.
  */}}

{{/* Create temporary copy of the top-level context: */}}
{{ $tplCtx := dict }}
{{ $_ := include "srox.mergeInto" (list $tplCtx $) }}
{{/* This is where we will store the iteratively constructed default values hierarchy: */}}
{{ $defaults := dict }}
{{ range $defaultsFile, $defaultsTpl := $.Files.Glob "internal/defaults/*.yaml" }}
  {{ $tplSects := regexSplit "(^|\n)---($|\n)" (toString $defaultsTpl) -1 }}
  {{ $sectCounter := 0 }}
  {{ range $tplSect := $tplSects }}
    {{/*
      tpl will merely stop creating output if an error is encountered during rendering (not during parsing), but we want
      to be certain that we recognized invalid templates. Hence, add a marker line at the end, and verify that it
      shows up in the output.
      */}}
    {{ $renderedSect := tpl (list $tplSect "{{ \"\\n#MARKER\\n\" }}" | join "") $tplCtx }}
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
    {{/* For maintaining our separate copy of default values: */}}
    {{ $_ = include "srox.mergeInto" (list $defaults $sectDict) }}
    {{/* For maintaining our separate copy of of the top-level context: */}}
    {{ $_ := include "srox.mergeInto" (list $tplCtx (dict "_rox" $sectDict)) }}
    {{ $sectCounter = add $sectCounter 1 }}
  {{ end }}
{{ end }}
{{ $_ := set $._rox "_defaults" $defaults }}
{{ $_ := set $._rox "_state" $tplCtx._rox._state }}
{{ end }}

{{/*
  srox.applyDefaults .

  Applies defaults defined in `internal/defaults`, in an order that depends on the filenames.
   */}}
{{ define "srox.applyDefaults" }}
{{ $ := . }}
{{ $_ := include "srox.mergeInto" (list $._rox $._rox._defaults) }}
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
