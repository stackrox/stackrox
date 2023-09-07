package events

import "github.com/stackrox/rox/pkg/sac/resources"

const (
	defaultRemediation = "An unknown issue occurred. Make sure to check out the detailed log view for more details."
)

var (
	hints = map[string]map[string]string{
		// Currently, a hint is based on the domain and the resource associated with an administration event.
		// In the future, we may extend this, and possibly also ensure hints are loaded externally (similar to
		// vulnerability definitions).
		imageScanningDomain: {
			// For now, this is an example string. We may want to revisit those together with UX / the docs team to get
			// errors that are in-line with documentation guidelines.
			resources.Image.String(): `An issue occurred scanning the image. Please ensure that:
- Scanner can access the registry.
- Correct credentials are configured for the particular registry / repository.
- The scanned manifest exists within the registry / repository.`,
		},
		defaultDomain: {},
	}
)

// GetHint retrieves the hint for a specific domain and resource.
// In case no hint exists for a specific area and resource, a generic text will be added.
//
// Currently, each hint is hardcoded and cannot be updated. In the future
// it might be possible to fetch definitions for a hint externally (e.g. similar to vulnerability definitions).
func GetHint(domain string, resource string) string {
	hintPerResource := hints[domain]
	if hintPerResource == nil {
		return defaultRemediation
	}

	hint := hintPerResource[resource]
	if hint == "" {
		return defaultRemediation
	}

	return hint
}
