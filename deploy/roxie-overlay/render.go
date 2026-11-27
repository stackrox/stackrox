package main

import (
	"fmt"
	"io"
	"text/template"
)

var imageTemplateKeys = map[image]string{
	mainImage:           "MainTag",
	centralDBImage:      "CentralDBTag",
	scannerV4Image:      "ScannerV4Tag",
	scannerV4DBImage:    "ScannerV4DBTag",
	collectorImage:      "CollectorTag",
	factImage:           "FactTag",
	operatorBundleImage: "OperatorTag",
}

func writeConfig(w io.Writer, registry string, tags map[image]string, base *BaseTag, overlayTag string) error {
	data := map[string]string{"ImageRegistry": registry, "OverlayTag": overlayTag}
	if baseResolved, ok := base.Peek(); ok {
		data["BaseTag"] = baseResolved
	}
	// Initialize for all images so that unresolved images (e.g. collector/fact without explicit tag)
	// render as empty, not "<no value>".
	for _, key := range imageTemplateKeys {
		data[key] = ""
	}
	for img, tag := range tags {
		data[imageTemplateKeys[img]] = tag
	}
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return fmt.Errorf("parsing config template: %w", err)
	}
	return tmpl.Execute(w, data)
}

// roxie.version selects the operator, so it comes from the resolved operator bundle. The operator
// defaults main-versioned operands to that version, so their overrides are only emitted when they
// differ. Collector/fact overrides are only emitted when given explicitly (see imageResolvers).
const configTemplate = `# roxie deployment config for deploying locally built images to a local cluster.
#
{{- if .BaseTag}}
#   Base version tag: {{.BaseTag}}
{{- end}}
#   Overlay version tag: {{.OverlayTag}}

roxie:
  imageRegistry: {{.ImageRegistry}}
  version: "{{.OperatorTag}}"
central:
  namespace: stackrox
  resourceProfile: auto
securedCluster:
  namespace: stackrox
  resourceProfile: auto
operator:
  operandTagOverrides:
{{- if ne .MainTag .OperatorTag}}
    main: "{{.MainTag}}"
{{- end}}
{{- if ne .CentralDBTag .OperatorTag}}
    central-db: "{{.CentralDBTag}}"
{{- end}}
{{- if ne .ScannerV4Tag .OperatorTag}}
    scanner-v4: "{{.ScannerV4Tag}}"
{{- end}}
{{- if ne .ScannerV4DBTag .OperatorTag}}
    scanner-v4-db: "{{.ScannerV4DBTag}}"
{{- end}}
{{- if .CollectorTag}}
    collector: "{{.CollectorTag}}"
{{- end}}
{{- if .FactTag}}
    fact: "{{.FactTag}}"
{{- end}}
`
