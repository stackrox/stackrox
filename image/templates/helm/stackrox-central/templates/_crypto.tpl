{{/*
  srox.configureCrypto $ $cryptoConfigPath $spec

  This helper function configures a private key or certificate (public cert + private key)
  config entry, from an input config which is accessed via $cryptoConfigPath relative to
  $._rox, which we'll refer to as $inputCfg. $inputCfg is expected to be a dict with at
  least `key` and `generate` properties. If `generate` is null, it defaults to either `true`
  on installations, and `false` on upgrades. `key` is an expandable string.
  The result in either mode is written to a dict $outputCfg under $._rox accessed by the
  $cryptoConfigPath, with a '_' prepended to the last path element. E.g., if
  $cryptoConfigPath is "a.b.c", the input configuration will be read from $._rox.a.b.c, and
  the output configuration will be stored in $._rox.a.b._c.

  Private key-only mode is selected if $spec.keyOnly contains a non-zero string, which specifies
  the key algorithm to use. In this mode, if $inputCfg.key expands to a non-empty string, this
  string will be copied to the `Key` property of $outputCfg. Otherwise, if $inputCfg.generate
  is true (wrt. the above defaulting rules), a key with the algorithm prescribed by $spec.keyOnly
  will be generated and stored in the `Key` property of $outputCfg.

  Certificate mode is the default. If $inputCfg.cert and $inputCfg.key expand to non-empty strings,
  these strings will be copied to the `Cert` and `Key` properties of $outputCfg. Otherwise, if both
  of them expand to empty strings (it is an error if only one of them expands to a non-empty
  string), and $inputCfg.generate is true, a certificate and private key are generated with the
  following options:
  - If $inputCfg.ca is true, generate a CA certificate with common name $inputCfg.CN and a 5 year
    validity duration.
  - Otherwise, generate a leaf certificate with common name $inputCfg.CN and a 1 year validity
    duration. The SANs for this certificate are derived from the base DNS name $inputCfg.dnsBase
    according to "srox.computeSANs".

  Whenever certificates and/or private keys were generated, the $._rox._state.generated property
  is updated to reflect the generated values, such that merging $._rox._state.generated in to
  $.Values would have caused this template to simply use the generated values as-is. E.g., if
  $cryptoConfigPath was "a.b.c" and $.Values.a.b.c.cert" and $.Values.a.b.c.key" were both empty,
  $._rox._state.generated.a.b.c would be set to be a dict with `cert` and `key` properties of the
  generated $outputCfg.Cert and $outputCfg.Key.

  If a certificate or private key was generated, $._rox._state.customCertGen is set to true.
   */}}
{{- define "srox.configureCrypto" -}}
{{ $ := index . 0 }}
{{ $cryptoConfigPath := index . 1 }}
{{ $spec := index . 2 }}

{{/* Resolve $cryptoConfigPath. */}}
{{ $cfg := $._rox }}
{{ $newGenerated := dict }}
{{ $genCfg := $newGenerated }}
{{ $cryptoConfigPathList := splitList "." $cryptoConfigPath }}
{{ range $pathElem := $cryptoConfigPathList }}
  {{ $cfg = index $cfg $pathElem }}
  {{ $newCfg := dict }}
  {{ $_ := set $genCfg $pathElem $newCfg }}
  {{ $genCfg = $newCfg }}
{{ end }}

{{/* Make sure `cert` and `key` are expanded (this should already be the case, but better
     safe than sorry. */}}
{{ $certExpandSpec := dict "cert" true "key" true }}
{{ include "srox.expandAll" (list $ $cfg $certExpandSpec $cryptoConfigPathList) }}

{{ $certPEM := $cfg._cert }}
{{ $keyPEM := $cfg._key }}

{{ $result := dict }}
{{ if and $certPEM $keyPEM }}
  {{ $result = dict "Cert" $certPEM "Key" $keyPEM }}
{{ else if or $certPEM $keyPEM }}
  {{ if and $keyPEM $spec.keyOnly }}
    {{ $_ := set $result "Key" $keyPEM }}
  {{ else }}
    {{ include "srox.fail" (printf "Either none or both of %s.cert and %s.key must be specified" $cryptoConfigPath $cryptoConfigPath) }}
  {{ end }}
{{ else }}
  {{ $generate := $cfg.generate }}
  {{ if kindIs "invalid" $generate }}
    {{ $generate = $.Release.IsInstall }}
  {{ end }}
  {{ if $generate }}
    {{ if $spec.ca }}
      {{ $result = genCA $spec.CN 1825 }}
    {{ else if $spec.keyOnly }}
      {{ $key := genPrivateKey $spec.keyOnly }}
      {{ $_ := set $genCfg "key" $key }}
      {{ $_ = set $result "Key" $key }}
    {{ else }}
      {{ if not $._rox._ca }}
        {{ include "srox.fail" (printf "Tried to generate certificate for %s, but no CA certificate is available." $spec.CN) }}
      {{ end }}
      {{ $sans := dict }}
      {{ include "srox.computeSANs" (list $ $sans $spec.dnsBase) }}
      {{ $ca := $._rox._ca }}
      {{ if kindIs "map" $ca }}
        {{ $ca = buildCustomCert (b64enc $ca.Cert) (b64enc $ca.Key) }}
      {{ end }}
      {{ $result = genSignedCert $spec.CN nil $sans.result 365 $ca }}
      {{ $_ := set $genCfg "cert" $result.Cert }}
      {{ $_ = set $genCfg "key" $result.Key }}
    {{ end }}
    {{ $_ := set $genCfg "key" $result.Key }}
    {{ if $result.Cert }}
      {{ $_ = set $genCfg "cert" $result.Cert }}
    {{ end }}
    {{ $_ = set $._rox._state "customCertGen" true }}
  {{ end }}
{{ end }}

{{/* Store output configuration and generated properties */}}
{{ $newCfgRoot := dict }}
{{ $newCfg := $newCfgRoot }}
{{ range $pathElem := initial $cryptoConfigPathList }}
  {{ $nextCfg := dict }}
  {{ $_ := set $newCfg $pathElem $nextCfg }}
  {{ $newCfg = $nextCfg }}
{{ end }}
{{ $_ := set $newCfg (last $cryptoConfigPathList | printf "_%s") $result }}
{{ $_ = include "srox.mergeInto" (list $._rox $newCfgRoot) }}
{{ $_ = include "srox.mergeInto" (list $._rox._state.generated $newGenerated) }}
{{ end }}


{{/*
  srox.configurePassword $ $pwConfigPath [$htpasswdUser]

  This helper function reads a password configuration (YAML dict with `value`
  and `generate` properties) referenced by $pwConfigPath relative to $._rox. It
  ensures the dict with the same config path relative to $._rox and prepending an underscore
  to the last path element is populated in the following way:
  - If the `value` property of the input config is nonzero, set `value` in the result to the
    expanded value.
  - If the optional $htpasswdUser parameter is specified and the `htpasswd` property of the
    input config is nonzero, set `htpasswd` in the result to the expanded value of that
    property.
  - If none of the above (non-mutually-exclusive) cases apply:
    - If `generate` is true OR both `generate` is null and this is an installation,
      not an upgrade, generate a random password with 32 alphanumeric characters.
    - Otherwise, leave the result property empty.
  - If the optional $htpasswdUser parameter was specified AND the `value` property in the
    result property was set per the above rules AND the `htpasswd` property was not set,
    populate the `htpasswd` property of the result by generating an htpasswd stanza with
    the computed `value` as the password and $htpasswdUser as the username.

  The $._rox._state.generated property is adjusted accordingly.
   */}}
{{- define "srox.configurePassword" -}}
{{ $ := index . 0 }}
{{ $pwConfigPath := index . 1 }}
{{ $htpasswdUser := "" }}
{{ if gt (len .) 2 }}
  {{ $htpasswdUser = index . 2 }}
{{ end }}
{{ $cfg := $._rox }}
{{ $newGenerated := dict }}
{{ $genCfg := $newGenerated }}
{{ $pwConfigPathList := splitList "." $pwConfigPath }}
{{ range $pathElem := $pwConfigPathList }}
  {{ $cfg = index $cfg $pathElem }}
  {{ $newCfg := dict }}
  {{ $_ := set $genCfg $pathElem $newCfg }}
  {{ $genCfg = $newCfg }}
{{ end }}

{{/* Make sure that `value` and `htpasswd` within $cfg are expanded (this should already be the
     case but better safe than sorry). */}}
{{ $pwExpandSpec := dict "value" true "htpasswd" true }}
{{ include "srox.expandAll" (list $ $cfg $pwExpandSpec $pwConfigPathList) }}

{{ $result := dict }}
{{ if and $htpasswdUser (not (kindIs "invalid" $cfg._htpasswd)) }}
  {{ $htpasswd := $cfg._htpasswd }}
  {{ $_ := set $result "htpasswd" $htpasswd }}
{{ end }}
{{ if not $result.htpasswd }}
  {{ $pw := dict.nil }}
  {{ if kindIs "invalid" $cfg._value }}
    {{ $generate := $cfg.generate }}
    {{ if kindIs "invalid" $generate }}
      {{ $generate = $.Release.IsInstall }}
    {{ end }}
    {{ if $generate }}
      {{ $pw = randAlphaNum 32 }}
      {{ $_ := set $genCfg "value" $pw }}
    {{ end }}
  {{ else }}
    {{ $pw = $cfg._value }}
  {{ end }}
  {{ if not (kindIs "invalid" $pw) }}
    {{ $_ := set $result "value" $pw }}
  {{ end }}
  {{ if and $htpasswdUser $pw }}
    {{ $htpasswd := htpasswd $htpasswdUser $pw }}
    {{ $_ := set $result "htpasswd" $htpasswd }}
  {{ end }}
{{ else if $cfg.value }}
  {{ include "srox.fail" (printf "Both a htpasswd and a value are specified for %s, this is illegal. Remove the `value` property, or ensure that `htpasswd` is null." $pwConfigPath) }}
{{ end }}
{{ $newCfgRoot := dict }}
{{ $newCfg := $newCfgRoot }}
{{ range $pathElem := initial $pwConfigPathList }}
  {{ $nextCfg := dict }}
  {{ $_ := set $newCfg $pathElem $nextCfg }}
  {{ $newCfg = $nextCfg }}
{{ end }}
{{ $_ := set $newCfg (last $pwConfigPathList | printf "_%s") $result }}
{{ $_ = include "srox.mergeInto" (list $._rox $newCfgRoot) }}
{{ $_ = include "srox.mergeInto" (list $._rox._state.generated $newGenerated) }}
{{ end }}


{{/*
  srox.computeSANs $ $out $svcName

  Compute the applicable SANs for a service with name $svcName, deployed in namespace
  $.Release.Namespace (= $releaseNS).
  Generally, SANs following the pattern "$svcName.$releaseNS[.svc[.cluster.local]]" will be
  generated. If $releaseNS is not "stackrox", another set of SANs with the same pattern,
  but assuming $releaseNS = "stackrox", will be generated in addition.
  The result is stored as a list in $out.result.
   */}}
{{ define "srox.computeSANs" }}
{{ $ := index . 0 }}
{{ $out := index . 1 }}
{{ $svcName := index . 2 }}
{{ $releaseNS := $.Release.Namespace }}
{{ $sans := list }}
{{ range $ns := list $releaseNS "stackrox" | uniq | sortAlpha }}
  {{ $baseDNS := printf "%s.%s" $svcName $ns }}
  {{ range $suffix := tuple "" ".svc" ".svc.cluster.local" }}
    {{ $sans = printf "%s%s" $baseDNS $suffix | append $sans }}
  {{ end }}
{{ end }}
{{ $_ := set $out "result" $sans }}
{{ end }}
