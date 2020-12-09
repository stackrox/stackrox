{{/*
  srox.configureImage $ $imageCfg

  Configures settings for a single image by augmenting/completing an existing image configuration
  stanza.

  If $imageCfg.fullRef is empty:
    First, the image registry is determined by inspecting $imageCfg.registry and, if this is empty,
    $._rox.image.registry, ultimately defaulting to `docker.io`. The full image ref is then
    constructed from the registry, $imageCfg.name (must be non-empty), and $imageCfg.tag (may be
    empty, in which case "latest" is assumed). The result is stored in $imageCfg.fullRef.

  Afterwards (irrespective of the previous check), $imageCfg.fullRef is modified by prepending
  "docker.io/" if and only if it did not contain a remote yet (i.e., the part before the first "/"
  did not contain a dot (DNS name) or colon (port)).

  Finally, the resulting $imageCfg.fullRef is stored as a dict entry with value `true` in the
  $._rox._state.referencedImages dict.
   */}}
{{ define "srox.configureImagePullSecrets" }}
{{ $ := index . 0 }}
{{ $cfgName := index . 1 }}
{{ $imagePullSecrets := index . 2 }}
{{ $defaultSecretNames := index . 3 }}

{{ $imagePullSecretNames := default list $imagePullSecrets.useExisting }}
{{ if not (kindIs "slice" $imagePullSecretNames) }}
  {{ $imagePullSecretNames = regexSplit "\\s*,\\s*" (trim $imagePullSecretNames) -1 }}
{{ end }}
{{ if $imagePullSecrets.useFromDefaultServiceAccount }}
  {{ $defaultSA := dict }}
  {{ include "srox.safeLookup" (list $ $defaultSA "v1" "ServiceAccount" $.Release.Namespace "default") }}
  {{ if $defaultSA.result }}
    {{ $imagePullSecretNames = concat $imagePullSecretNames (default list $defaultSA.result.imagePullSecrets) }}
  {{ end }}
{{ end }}
{{ $imagePullCreds := dict }}
{{ if $imagePullSecrets._username }}
  {{ $imagePullCreds = dict "username" $imagePullSecrets._username "password" $imagePullSecrets._password }}
  {{ $imagePullSecretNames = append $imagePullSecretNames "stackrox" }}
{{ else if $imagePullSecrets._password }}
  {{ $msg := printf "Password missing in %q. Whenever an image pull password is specified, a username must be specified as well" $cfgName }}
  {{ include "srox.fail" }}
{{ end }}
{{ if and $.Release.IsInstall (not $imagePullSecretNames) (not $imagePullSecrets.allowNone) }}
  {{ $msg := printf "You have not specified any image pull secrets, and no existing image pull secrets were automatically inferred. If your registry does not need image pull credentials, explicitly set the '%s.allowNone' option to 'true'" $cfgName }}
  {{ include "srox.fail" $msg }}
{{ end }}

{{/*
    Always assume that there are `stackrox` and `stackrox-scanner` image pull secrets,
    even if they weren't specified.
    This is required for updates anyway, so referencing it on first install will minimize a later
    diff.
   */}}
{{ $imagePullSecretNames = concat $imagePullSecretNames $defaultSecretNames | uniq | sortAlpha }}
{{ $_ := set $imagePullSecrets "_names" $imagePullSecretNames }}
{{ $_ := set $imagePullSecrets "_creds" $imagePullCreds }}

{{ end }}

{{ define "srox.configureImagePullSecretsForDockerRegistry" }}
{{ $ := index . 0 }}
{{ $imagePullSecrets := index . 1 }}

{{/* Setup Image Pull Secrets for Docker Registry.
     Note: This must happen afterwards, as we rely on "srox.configureImage" to collect the
     set of all referenced images first. */}}
{{ if $imagePullSecrets._username }}
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
{{ end }}

{{ end }}

