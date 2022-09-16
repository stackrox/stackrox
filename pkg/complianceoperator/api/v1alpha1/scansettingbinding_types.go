package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamedObjectReference struct {
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	APIGroup string `json:"apiGroup,omitempty"`
}

// +kubebuilder:object:root=true

// ScanSettingBinding is the Schema for the scansettingbindings API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scansettingbindings,scope=Namespaced,shortName=ssb
type ScanSettingBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Profiles    []NamedObjectReference `json:"profiles,omitempty"`
	SettingsRef *NamedObjectReference  `json:"settingsRef,omitempty"`
	// +optional
	Status ScanSettingBindingStatus `json:"status,omitempty"`
}

type ScanSettingBindingStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Reference to the object generated from this ScanSettingBinding
	// +optional
	// +nullable
	OutputRef *corev1.TypedLocalObjectReference `json:"outputRef,omitempty"`
}

// +kubebuilder:object:root=true

// ScanSettingBindingList contains a list of ScanSettingBinding
type ScanSettingBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScanSettingBinding `json:"items"`
}

func (s *ScanSettingBindingStatus) SetConditionPending() {
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "Pending",
		Message: "The scan setting binding is waiting to be processed",
	})
}

func (s *ScanSettingBindingStatus) SetConditionInvalid(msg string) {
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "Invalid",
		Message: msg,
	})
}

func (s *ScanSettingBindingStatus) SetConditionReady() {
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "Processed",
		Message: "The scan setting binding was successfully processed",
	})
}

func init() {
	SchemeBuilder.Register(&ScanSettingBinding{}, &ScanSettingBindingList{})
}
