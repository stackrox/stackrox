package common

import (
	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// These fields of SecurityContextConstraints are advertised as required in the OpenAPI schema, but they actually
	// aren't.
	sccNonRequired = set.NewFrozenStringSet(
		// The following fields will be reported as `null` by the server if they are empty/unset, but a `null` value
		// would fail schema validation ... oO.
		"allowedCapabilities",
		"defaultAddCapabilities",
		"requiredDropCapabilities",
	)
)

// patchOpenAPISchema modifies the OpenAPI schema to fix some issues, particularly on OpenShift.
func patchOpenAPISchema(doc *openapi_v2.Document) error {
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
