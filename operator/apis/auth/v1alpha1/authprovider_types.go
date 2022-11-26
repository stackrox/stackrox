/*
Copyright 2021 Red Hat.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthProviderSpec defines the desired state of AuthProvider
type AuthProviderSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type allows you to specify the specific auth provider you want to create.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Type,order=1
	Type *AuthProviderType `json:"type,omitempty"`

	// ClientID allows you to specify the client ID for the OIDC client.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Client ID,order=2
	ClientID *string `json:"clientId,omitempty"`

	// ClientSecretReference allows you to specify an optional secret that holds the client secret for the OIDC client.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Client Reference,order=3
	ClientSecretReference *SecretReference `json:"clientSecretReference,omitempty"`

	// Issuer allows you to specify the issuer of the OIDc client.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Issuer,order=3
	Issuer *string `json:"issuer,omitempty"`
}

// AuthProviderType specifies the type of the auth provider.
type AuthProviderType string

const (
	// AuthProviderOIDC means that the created auth provider will be using OIDC.
	AuthProviderOIDC AuthProviderType = "oidc"
)

// SecretReference is a reference to a secret within the same namespace.
type SecretReference struct {
	// Name of the referenced secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	Name string `json:"name"`
}

// AuthProviderStatus defines the observed state of AuthProvider
type AuthProviderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AuthProvider is the Schema for the authproviders API
type AuthProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthProviderSpec   `json:"spec,omitempty"`
	Status AuthProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AuthProviderList contains a list of AuthProvider
type AuthProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthProvider{}, &AuthProviderList{})
}
