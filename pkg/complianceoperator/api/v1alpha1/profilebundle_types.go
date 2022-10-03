package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProfileBundleFinalizer is a finalizer for ProfileBundles. It gets automatically
// added by the ProfileBundle controller in order to delete resources.
const ProfileBundleFinalizer = "profilebundle.finalizers.compliance.openshift.io"

// ProfileBundleOwnerLabel marks a profile or rule as owned by a profile bundle
// and helps users filter such objects
const ProfileBundleOwnerLabel = "compliance.openshift.io/profile-bundle"

// ProfileImageDigestAnnotation is the parsed out digest of the content image
const ProfileImageDigestAnnotation = "compliance.openshift.io/image-digest"

// DataStreamStatusType is the type for the data stream status
type DataStreamStatusType string

const (
	// DataStreamPending represents the state where the data stream
	// hasn't been processed yet
	DataStreamPending DataStreamStatusType = "PENDING"
	// DataStreamValid represents the status for a valid data stream
	DataStreamValid DataStreamStatusType = "VALID"
	// DataStreamInvalid represents the status for an invalid data stream
	DataStreamInvalid DataStreamStatusType = "INVALID"
)

// Defines the desired state of ProfileBundle
type ProfileBundleSpec struct {
	// Is the path for the image that contains the content for this bundle.
	ContentImage string `json:"contentImage"`
	// Is the path for the file in the image that contains the content for this bundle.
	ContentFile string `json:"contentFile"`
}

// Defines the observed state of ProfileBundle
type ProfileBundleStatus struct {
	// Presents the current status for the datastream for this bundle
	// +kubebuilder:default=PENDING
	DataStreamStatus DataStreamStatusType `json:"dataStreamStatus,omitempty"`
	// If there's an error in the datastream, it'll be presented here
	ErrorMessage string `json:"errorMessage,omitempty"`
	// Defines the conditions for the ProfileBundle. Valid conditions are:
	//  - Ready: Indicates if the ProfileBundle is Ready parsing or not.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// ProfileBundle is the Schema for the profilebundles API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=profilebundles,scope=Namespaced,shortName=pb
// +kubebuilder:printcolumn:name="ContentImage",type="string",JSONPath=`.spec.contentImage`
// +kubebuilder:printcolumn:name="ContentFile",type="string",JSONPath=`.spec.contentFile`
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=`.status.dataStreamStatus`
type ProfileBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProfileBundleSpec   `json:"spec,omitempty"`
	Status ProfileBundleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProfileBundleList contains a list of ProfileBundle
type ProfileBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProfileBundle `json:"items"`
}

func (s *ProfileBundleStatus) SetConditionPending() {
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "Pending",
		Message: "The profile bundle is waiting to be parsed",
	})
}

func (s *ProfileBundleStatus) SetConditionInvalid() {
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "Invalid",
		Message: "Couldn't parse profile bundle",
	})
}

func (s *ProfileBundleStatus) SetConditionReady() {
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "Valid",
		Message: "Profile bundle successfully parsed",
	})
}

func init() {
	SchemeBuilder.Register(&ProfileBundle{}, &ProfileBundleList{})
}
