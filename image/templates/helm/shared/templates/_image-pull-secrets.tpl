{{/*
  srox.configureImagePullSecrets $ $cfgName $imagePullSecrets $secretResourceName $defaultSecretNames $namespace

  Configures image pull secrets.

  Note: This function must be called late in srox.init, as we rely on "srox.configureImage" to collect the
  set of all referenced images first.

  This function enriches $imagePullSecrets based on the exposed configuration parameters to contain:

  1. Optionally (i.e. only if $imagePullSecrets contains a username), a `_dockerAuths` field, containing a map
     from registry URLs to settings that contain login credentials.
     The map contains registries for all images passed so far to srox.configureImage invocations.

  2. A `_names` field, containing a list of Kubernetes secret names. The chart templates then use this field
     to populate imagePullSecrets lists in ServiceAccount objects.

     The list contains the following secrets:

     - Secrets referenced via $imagePullSecrets.useExisting.
     - Image pull secrets associated with the default service account (unless
       $imagePullSecrets.useFromDefaultServiceAccount was set to false by the user).
     - $secretResourceName.
     - $defaultSecretNames.

  Additionally, this function fails execution if the list resulting from first three bullet points
  combined is empty.

*/}}

{{ define "srox.configureImagePullSecrets" }}
{{ $ := index . 0 }}
{{ $cfgName := index . 1 }}
{{ $imagePullSecrets := index . 2 }}
{{ $secretResourceName := index . 3 }}
{{ $defaultSecretNames := index . 4 }}
{{ $namespace := index . 5 }}

{{ $imagePullSecretNames := default list $imagePullSecrets.useExisting }}
{{ if not (kindIs "slice" $imagePullSecretNames) }}
  {{ $imagePullSecretNames = regexSplit "\\s*[,;]\\s*" (trim $imagePullSecretNames) -1 }}
{{ end }}

{{ if $imagePullSecrets.useFromDefaultServiceAccount }}
  {{ $defaultSA := dict }}
  {{ include "srox.safeLookup" (list $ $defaultSA "v1" "ServiceAccount" $namespace "default") }}
  {{ if $defaultSA.result }}
    {{ range $ips := default list $defaultSA.result.imagePullSecrets }}
      {{ if $ips.name }}
        {{ $imagePullSecretNames = append $imagePullSecretNames $ips.name }}
      {{ end }}
    {{ end }}
  {{ end }}
{{ end }}

{{ if $imagePullSecrets._username }}
  {{/* When username is present, existence of $secretResourceName will be assured by the templates; add to the list. */}}
  {{ $imagePullSecretNames = append $imagePullSecretNames $secretResourceName }}

  {{ $dockerAuths := dict }}
  {{ range $image := keys $._rox._state.referencedImages }}
    {{ $registry := splitList "/" $image | first }}
    {{ if eq $registry "docker.io" }}
      {{/* Special case docker.io */}}
      {{ $registry = "https://index.docker.io/v1/" }}
    {{ else }}
      {{ $registry = printf "https://%s" $registry }}
    {{ end }}
    {{ $_ := set $dockerAuths $registry dict }}
  {{ end }}
  {{ $authToken := printf "%s:%s" $imagePullSecrets._username $imagePullSecrets._password | b64enc }}
  {{ range $regSettings := values $dockerAuths }}
    {{ $_ := set $regSettings "auth" $authToken }}
  {{ end }}

  {{ $_ := set $imagePullSecrets "_dockerAuths" $dockerAuths }}

{{ else if $imagePullSecrets._password }}
  {{ $msg := printf "Username missing in %q. Whenever an image pull password is specified, a username must be specified as well" $cfgName }}
  {{ include "srox.fail" $msg }}
{{ end }}

{{ if and $.Release.IsInstall (not $imagePullSecretNames) (not $imagePullSecrets.allowNone) }}
  {{ $msg := printf "You have not specified any image pull secrets, and no existing image pull secrets were automatically inferred. If your registry does not need image pull credentials, explicitly set the '%s.allowNone' option to 'true'" $cfgName }}
  {{ include "srox.fail" $msg }}
{{ end }}

{{ $imagePullSecretNames = concat (append $imagePullSecretNames $secretResourceName) $defaultSecretNames | uniq | sortAlpha }}
{{ $_ := set $imagePullSecrets "_names" $imagePullSecretNames }}

{{ end }}
