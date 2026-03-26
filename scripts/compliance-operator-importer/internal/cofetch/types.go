// Package cofetch defines types for Compliance Operator resource discovery.
package cofetch

import "context"

// NamedObjectReference is a lightweight reference to a named Kubernetes object.
// It mirrors the CO NamedObjectReference type without importing the CO library.
type NamedObjectReference struct {
	Name     string
	Kind     string // "Profile" or "TailoredProfile"; empty defaults to "Profile" (IMP-MAP-002)
	APIGroup string
}

// ResolvedKind returns the kind, defaulting to "Profile" when empty (IMP-MAP-002).
func (r NamedObjectReference) ResolvedKind() string {
	if r.Kind == "" {
		return "Profile"
	}
	return r.Kind
}

// ProfileRef is an alias for NamedObjectReference used in profile reference lists.
// It is a type alias (not a new type) so []ProfileRef and []NamedObjectReference are
// interchangeable, allowing both client.go and mapping_test.go to construct profiles.
type ProfileRef = NamedObjectReference

// ScanSettingBinding is a simplified representation of the Compliance Operator
// ScanSettingBinding resource (compliance.openshift.io/v1alpha1).
// Fields are extracted from unstructured Kubernetes API responses.
type ScanSettingBinding struct {
	Namespace       string
	Name            string
	ScanSettingName string                // name of the referenced ScanSetting (flattened from SettingsRef.Name)
	SettingsRef     *NamedObjectReference // full structured settings reference
	Profiles        []NamedObjectReference
}

// ScanSetting is a simplified representation of the Compliance Operator ScanSetting
// resource (compliance.openshift.io/v1alpha1).
type ScanSetting struct {
	Namespace string
	Name      string
	// Schedule is the cron expression from complianceSuiteSettings.schedule.
	Schedule string
}

// COClient abstracts Compliance Operator resource discovery.
type COClient interface {
	// ListScanSettingBindings returns all ScanSettingBindings in the configured namespace(s).
	ListScanSettingBindings(ctx context.Context) ([]ScanSettingBinding, error)
	// GetScanSetting fetches a named ScanSetting from the given namespace.
	GetScanSetting(ctx context.Context, namespace, name string) (*ScanSetting, error)
	// PatchSSBSettingsRef patches the settingsRef.name of a ScanSettingBinding.
	PatchSSBSettingsRef(ctx context.Context, namespace, ssbName, newSettingsRefName string) error
}
