package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	GroupName = "authprovider.stackrox.io"
	Kind      = "AuthProvider"
	Version   = "v1beta1"
	Plural    = "authproviders"
	Singular  = "authprovider"
	ShortName = "ap"
	Name      = Plural + "." + GroupName
)

// AuthProviderSpec is the config of an auth provider
type AuthProviderSpec struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	UiEndpoint string            `json:"uiEndpoint"`
	Enabled    bool              `json:"enabled"`
	Config     map[string]string `json:"config"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +noStatus
// TODO: Add ,k8s.io/kubernetes/runtime.List generation above?

// AuthProvider represents an auth provider in the kubernetes API
type AuthProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthProviderSpec `json:"spec"`
	Status string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// AuthProviderList is a list of AuthProvider resources
type AuthProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthProvider `json:"items"`
}
