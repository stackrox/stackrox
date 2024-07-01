# This file contains defaults that need to be merged into our config struct before we can
# execute the "normal" defaulting logic. As a result, none of these values can be overridden
# by defaults specified in defaults.yaml and platforms/*.yaml - that is okay.

{{- if eq .Release.Name "test-release" }}
{{- include "srox.warn" (list . "You are using a release name that is reserved for tests. In order to allow linting to work, certain checks have been relaxed. If you are deploying to a real environment, we recommend that you choose a different release name.") }}
allowNonstandardNamespace: true
allowNonstandardReleaseName: true
{{- else }}
allowNonstandardNamespace: false
allowNonstandardReleaseName: false
{{- end }}

meta:
  useLookup: true
  fileOverrides: {}
