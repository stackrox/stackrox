package translation

import (
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

// ToHelmValues returns the given object (which must be a (pointer to a) map or struct)
// and converts it to a chartutil.Values representation that can be used for Helm templating.
// The Helm rendering engine passes the chartutil.Values directly to the templating engine.
// This means that even if the YAML looks fine, the Helm template might fail in unexpected ways
// because currently map keys are no longer of type string, but instead of some user-defined
// type.
// We do a marshal/unmarshal round trip (using the Kubernetes apimachinery for marshaling, and Helm
// for unmarshaling) to ensure that the representation is exactly as during a normal Helm invocation.
func ToHelmValues(v interface{}) (chartutil.Values, error) {
	dataAsYaml, err := yaml.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling values")
	}
	values, err := chartutil.ReadValues(dataAsYaml)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling previously marshaled values")
	}
	return values, nil
}
