package search

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/set"
)

// GlobalOptions is exposed for e2e test
var GlobalOptions = []string{
	"Cluster",
	"Label",
	"Namespace",
}

// PolicyOptionsMap is exposed for e2e test
var PolicyOptionsMap = map[string]string{
	//"Enforcement": "policy.enforcement", // removing for now due to not handling enums properly
	"Policy Name": "policy.name",
	"Description": "policy.description",
	"Category":    "policy.categories",
}

// ImageOptionsMap is exposed for e2e test
var ImageOptionsMap = map[string]string{
	"CVE":                    "image.scan.components.vulns.cve",
	"Component":              "image.scan.components.name",
	"Dockerfile Instruction": "image.metadata.layers.instruction",
	"Image Name":             "image.name.fullname",
	"Image Registry":         "image.name.registry",
	"Image Namespace":        "image.name.namespace",
	"Image Repo":             "image.name.repo",
	"Image Tag":              "image.name.tag",
}

// DeploymentOptionsMap is exposed for e2e test
var DeploymentOptionsMap = map[string]string{
	"Add Capabilities":  "deployment.containers.security_context.add_capabilities",
	"Deployment Name":   "deployment.name",
	"Deployment Type":   "deployment.type",
	"Drop Capabilities": "deployment.containers.security_context.drop_capabilities",
	"Environment Key":   "deployment.containers.config.env.key",
	"Environment Value": "deployment.containers.config.env.value",
	//"Privileged":         "deployment.containers.security_context.privileged", // Need to add mapping AP-490
	"Volume Name":        "deployment.containers.volumes.name",
	"Volume Source":      "deployment.containers.volumes.source",
	"Volume Destination": "deployment.containers.volumes.destination",
	//"Volume ReadOnly":    "deployment.containers.volumes.read_only", // Need to add mapping
	"Volume Type": "deployment.containers.volumes.type",
}

// allOptionsMaps is the list of all options
var allOptionsMaps = map[string]string{
	// Alert Options
	"Alert Name": "alert.policy.name",
	"Violation":  "alert.violations.message",

	// PolicyOptions
	"Policy Name": "policy.name",
	"Description": "policy.description",
	"Category":    "policy.categories",

	// ImageOptions
	"CVE":                    "image.scan.components.vulns.cve",
	"Component":              "image.scan.components.name",
	"Dockerfile Instruction": "image.metadata.layers.instruction",
	"Image Name":             "image.name.full_name",
	"Image Registry":         "image.name.registry",
	"Image Namespace":        "image.name.namespace",
	"Image Repo":             "image.name.repo",
	"Image Tag":              "image.name.tag",

	// DeploymentOptions
	"Add Capabilities":   "deployment.containers.security_context.add_capabilities",
	"Deployment Name":    "deployment.name",
	"Deployment Type":    "deployment.type",
	"Drop Capabilities":  "deployment.containers.security_context.drop_capabilities",
	"Environment Key":    "deployment.containers.config.env.key",
	"Environment Value":  "deployment.containers.config.env.value",
	"Volume Name":        "deployment.containers.volumes.name",
	"Volume Source":      "deployment.containers.volumes.source",
	"Volume Destination": "deployment.containers.volumes.destination",
	"Volume Type":        "deployment.containers.volumes.type",
}

// AlertOptionsMap is exposed for e2e test
var AlertOptionsMap = map[string]string{
	"Alert Name": "alert.policy.name",
	"Violation":  "alert.violations.message",
}

// GetOptions returns the searchable fields for the specified categories
func GetOptions(categories []v1.SearchCategory) []string {
	optionsSet := set.NewSetFromStringSlice(GlobalOptions)
	for _, category := range categories {
		switch category {
		case v1.SearchCategory_ALERTS:
			set.AppendStringMapKeys(optionsSet, AlertOptionsMap)
			set.AppendStringMapKeys(optionsSet, PolicyOptionsMap)
			set.AppendStringMapKeys(optionsSet, DeploymentOptionsMap)
			set.AppendStringMapKeys(optionsSet, ImageOptionsMap)
		case v1.SearchCategory_POLICIES:
			set.AppendStringMapKeys(optionsSet, PolicyOptionsMap)
		case v1.SearchCategory_DEPLOYMENTS:
			set.AppendStringMapKeys(optionsSet, DeploymentOptionsMap)
			set.AppendStringMapKeys(optionsSet, ImageOptionsMap)
		case v1.SearchCategory_IMAGES:
			set.AppendStringMapKeys(optionsSet, ImageOptionsMap)
		}
	}
	slice := set.StringSliceFromSet(optionsSet)
	sort.Strings(slice)
	return slice
}
