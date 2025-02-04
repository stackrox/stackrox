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
{{ define "srox.configureImage" }}
{{ $ := index . 0 }}
{{ $imageCfg := index . 1 }}
{{ $imageRef := $imageCfg.fullRef }}
{{ if not $imageRef }}
  {{ $imageRef = printf "%s/%s:%s" (coalesce $imageCfg.registry $._rox.image.registry "docker.io") $imageCfg.name (default "latest" $imageCfg.tag) }}
{{ end }}
{{ $imageComponents := splitList "/" $imageRef }}
{{ $firstComponent := index $imageComponents 0 }}
{{ if or (lt (len $imageComponents) 2) (and (not (contains ":" $firstComponent)) (not (contains "." $firstComponent))) }}
  {{ $imageRef = printf "docker.io/%s" $imageRef }}
{{ end }}
{{ $_ := set $imageCfg "fullRef" $imageRef }}
{{ $_ = set $._rox._state.referencedImages $imageRef true }}
{{ end }}
