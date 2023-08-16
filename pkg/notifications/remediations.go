package notifications

const (
	defaultRemediation = "An unknown issue occurred. Make sure to check out the detailed log view for more details."
)

var (
	staticRemediations = map[string]map[string]string{
		// What will be the best key? The specific file, or rather the area? I suppose we have to do:
		// Figure out the area, map the area based on the error (and maybe also the resource involved.
		"Image scanning": {

		},
		"General":        {},
	}
)

// GetRemediation retrieves the remediation for a specific area and resource.
// In case no remediation exists for a specific area and resource, a generic text will be added.
//
// Currently, each remediation is hardcoded and cannot be updated. In the future
// it might be possible to fetch definitions for a remediation externally (e.g. similar to vulnerability definitions).
func GetRemediation(area string, resource string) string {
	remediationPerResource := staticRemediations[area]
	if remediationPerResource == nil {
		return defaultRemediation
	}

	remediation := remediationPerResource[resource]
	if remediation == "" {
		return defaultRemediation
	}


	resources.

	return remediation
}
