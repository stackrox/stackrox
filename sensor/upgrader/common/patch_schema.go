package common

import (
	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// These fields of SecurityContextConstraints are advertised as required in the OpenAPI schema, but they actually
	// aren't.
	sccNonRequired = set.NewFrozenStringSet(
		"allowPrivilegedContainer",
		"defaultAddCapabilities",
		"requiredDropCapabilities",
		"allowedCapabilities",
		"allowHostDirVolumePlugin",
		"volumes",
		"allowHostNetwork",
		"allowHostPorts",
		"allowHostPID",
		"allowHostIPC",
		"readOnlyRootFilesystem",
	)
)

// PatchOpenAPISchema modifies the OpenAPI schema to fix some issues, particularly on OpenShift.
func PatchOpenAPISchema(doc *openapi_v2.Document) error {
	for _, def := range doc.GetDefinitions().GetAdditionalProperties() {
		if def.GetName() != "com.github.openshift.api.security.v1.SecurityContextConstraints" {
			continue
		}
		req := def.GetValue().GetRequired()
		newReq := req[:0]

		for _, field := range req {
			if sccNonRequired.Contains(field) {
				continue
			}
			newReq = append(newReq, field)
		}
		def.Value.Required = newReq
	}
	return nil
}
