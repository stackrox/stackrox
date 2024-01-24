package complianceoperator

import (
	compv1alpha1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Compliance operator API set.
	groupVersion = compv1alpha1.SchemeGroupVersion

	// List of required compliance operator CRDs.
	requiredAPIResources []APIResource
)

// APIResources for compliance operator resources.
var (
	Profile = registerAPIResource(v1.APIResource{
		Name:    "profiles",
		Kind:    "Profile",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	Rule = registerAPIResource(v1.APIResource{
		Name:    "rules",
		Kind:    "Rule",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	ScanSetting = registerAPIResource(v1.APIResource{
		Name:    "scansettings",
		Kind:    "ScanSetting",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	ScanSettingBinding = registerAPIResource(v1.APIResource{
		Name:    "scansettingbindings",
		Kind:    "ScanSettingBinding",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	ComplianceScan = registerAPIResource(v1.APIResource{
		Name:    "compliancescans",
		Kind:    "ComplianceScan",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	ComplianceSuite = registerAPIResource(v1.APIResource{
		Name:    "compliancesuites",
		Kind:    "ComplianceSuite",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	ComplianceCheckResult = registerAPIResource(v1.APIResource{
		Name:    "compliancecheckresults",
		Kind:    "ComplianceCheckResult",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	TailoredProfile = registerAPIResource(v1.APIResource{
		Name:    "tailoredprofiles",
		Kind:    "TailoredProfile",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
)

// GetGroupVersion return the group version that uniquely represents the API set of compliance operator CRs.
func GetGroupVersion() schema.GroupVersion {
	return groupVersion
}

// GetRequiredResources returns the compliance operator API resources required by ACS.
func GetRequiredResources() []APIResource {
	return requiredAPIResources
}

// APIResource provides a wrapper around v1.APIResource.
type APIResource struct {
	v1.APIResource
}

// GroupVersionKind returns the GroupVersionKind which uniquely identifies the resource kind.
func (r *APIResource) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   r.Kind,
		Version: r.Version,
		Kind:    r.Kind,
	}
}

// GroupVersionResource returns the GroupVersionResource which uniquely identifies the resource.
func (r *APIResource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Group,
		Version:  r.Version,
		Resource: r.Name,
	}
}

func registerAPIResource(resource v1.APIResource) APIResource {
	r := APIResource{resource}
	requiredAPIResources = append(requiredAPIResources, r)
	return r
}
