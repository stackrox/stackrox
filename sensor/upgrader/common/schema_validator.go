package common

import (
	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	"github.com/pkg/errors"
	"k8s.io/kubectl/pkg/util/openapi"
	"k8s.io/kubectl/pkg/validation"
	openAPIValidation "k8s.io/kubectl/pkg/validation"
)

// ValidatorFromOpenAPIDoc takes a given OpenAPI v2 Document and returns a schema validator for it.
func ValidatorFromOpenAPIDoc(openAPIDoc *openapi_v2.Document) (validation.Schema, error) {
	if err := patchOpenAPISchema(openAPIDoc); err != nil {
		return nil, errors.Wrap(err, "patching OpenAPI schema")
	}
	openAPIResources, err := openapi.NewOpenAPIData(openAPIDoc)
	if err != nil {
		return nil, errors.Wrap(err, "parsing OpenAPI schema document into resources")
	}
	schemaValidator := openAPIValidation.NewSchemaValidation(openAPIResources)

	return validation.ConjunctiveSchema{
		schemaValidator,
		yamlValidator{jsonValidator: validation.NoDoubleKeySchema{}},
	}, nil
}
