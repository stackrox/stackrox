package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// ScanSetting is the Schema for the scansettings API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scansettings,scope=Namespaced,shortName=ss
type ScanSetting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ComplianceSuiteSettings `json:",inline"`
	ComplianceScanSettings  `json:",inline"`
	// The list of roles to apply node-specific checks to
	Roles []string `json:"roles,omitempty"`
}

// +kubebuilder:object:root=true

// ScanSettingList contains a list of ScanSetting
type ScanSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScanSetting `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScanSetting{}, &ScanSettingList{})
}
