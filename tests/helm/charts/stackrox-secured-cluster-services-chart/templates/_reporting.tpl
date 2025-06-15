{{/*
  srox.fail $message

  Print a nicely-formatted fatal error message and exit.
   */}}
{{ define "srox.fail" }}
{{ printf "\n\nFATAL ERROR:\n%s" . | wrap 100 | fail }}
{{ end }}

{{/*
  srox.warn $ $message

  Add $message to the list of encountered warnings.
   */}}
{{ define "srox.warn" }}
{{ $ := index . 0 }}
{{ $msg := index . 1 }}
{{ $warnings := $._rox._state.warnings }}
{{ $warnings = append $warnings $msg }}
{{ $_ := set $._rox._state "warnings" $warnings }}
{{ end }}

{{/*
  srox.note $ $message

  Add $message to the list notes that will be shown to the user after installation/upgrade.
   */}}
{{ define "srox.note" }}
{{ $ := index . 0 }}
{{ $msg := index . 1 }}
{{ $notes := $._rox._state.notes }}
{{ $notes = append $notes $msg }}
{{ $_ := set $._rox._state "notes" $notes }}
{{ end }}
