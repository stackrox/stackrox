/*
Copyright 2021.

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
	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make generate manifests" to regenerate code after modifying this file
// TODO(ROX-7110): prevent merging PRs if manifests are not up to date.

// -------------------------------------------------------------
// Spec

// CentralSpec defines the desired state of Central
type CentralSpec struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Egress *Egress `json:"egress,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	TLS *common.TLSConfig `json:"tls,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Central *CentralComponentSpec `json:"central,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Scanner *ScannerComponentSpec `json:"scanner,omitempty"`
	// Customizations to apply on all central components.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Customize *common.CustomizeSpec `json:"customize,omitempty"`
}

// Egress defines settings related to outgoing network traffic.
type Egress struct {
	ConnectivityPolicy *ConnectivityPolicy `json:"connectivityPolicy,omitempty"`
	// TODO(ROX-7272): support proxy-aware openshift infrastructure feature

	// Reference to a secret which must contain a member named "config.yaml" that specifies the proxy configuration for central and scanner.
	ProxyConfigSecret *corev1.LocalObjectReference `json:"proxyConfigSecret,omitempty"`
}

// ConnectivityPolicy is a type for values of spec.egress.connectivityPolicy.
type ConnectivityPolicy string

const (
	// ConnectivityOnline means that Central is allowed to make outbound connections to the Internet.
	ConnectivityOnline ConnectivityPolicy = "Online"
	// ConnectivityOffline means that Central must not make outbound connections to the Internet.
	ConnectivityOffline ConnectivityPolicy = "Offline"
)

// CentralComponentSpec defines settings for the "central" component.
type CentralComponentSpec struct {
	common.DeploymentSpec `json:",inline"`
	TelemetryPolicy       *TelemetryPolicy     `json:"telemetryPolicy,omitempty"`
	Endpoint              *CentralEndpointSpec `json:"endpoint,omitempty"`
	Crypto                *CentralCryptoSpec   `json:"crypto,omitempty"`

	// Implementation note: this is distinct from the secret that contains the htpasswd-encoded password mounted in central.
	// TODO(ROX-7242): expose the secret name unconditionally

	// Reference to a user-created Secret with admin password stored in data item "value".
	// If omitted, the operator will instead auto-generate a password, create such secret and
	// expose the name of that secret (if it is different from the name "central-admin-password") in status.central.generatedAdminPasswordSecret
	AdminPasswordSecret *corev1.LocalObjectReference `json:"adminPasswordSecret,omitempty"`
	Persistence         *Persistence                 `json:"persistence,omitempty"`
	Exposure            *Exposure                    `json:"exposure,omitempty"`
	// TODO(ROX-7123): determine whether we want to make `extraMounts` available in the operator
	// TODO(ROX-7112): should we expose central.config? It's exposed in helm charts but not documented in help.stackrox.com.
}

// TelemetryPolicy is a type for values of spec.centralSpec.telemetryPolicy.
type TelemetryPolicy string

const (
	// TelemetryEnabled means that telemetry should be enabled.
	TelemetryEnabled TelemetryPolicy = "Enabled"
	// TelemetryDisabled means that telemetry should be disabled.
	TelemetryDisabled TelemetryPolicy = "Disabled"
)

// CentralEndpointSpec defines the endpoint config for central.
type CentralEndpointSpec struct {
	// TODO(ROX-7147): design this
	// should this be an opaque YAML like in helm or structured data that would let us configure
	// network policy as well? I.e. should this be merged with Exposure?
}

// CentralCryptoSpec defines custom crypto-related settings for central.
type CentralCryptoSpec struct {
	// TODO(ROX-7148): design this
	// this should configure the following helm values:
	// - central.jswSigner
	// - central.serviceTLS (potentially; see DeploymentSpec.ServiceTLS)
	// - central.defaultTLS
	// - ca (potentially, see below)

	// AFAICT the helm chart puts them all (including stuff from common.TLSConfig.CASecret) into a single
	// secret resource for consumption by central. For the operator we could:
	// - allow the to specify individual secret resources, consumed by
	//   central directly (best UX, but would likely require app code changes), or
	// - allow the to specify individual secret resources, consumed only by
	//   the operator to assemble into a single secret in turn read by central
	//   (no central changes but needs some code in operator), or
	// - allow user to specify a single secret in an all-or-nothing manner,
	//   which would be directly read by unmodified central (worst UX, but least dev effort)
}

// Persistence defines persistence settings for central.
type Persistence struct {
	HostPath              *string                `json:"hostPath,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`
}

// PersistentVolumeClaim defines PVC-based persistence settings.
type PersistentVolumeClaim struct {
	ClaimName   *string            `json:"claimName,omitempty"`
	CreateClaim *ClaimCreatePolicy `json:"createClaim,omitempty"`
	// TODO(ROX-7149): more details TBD, values files are inconsistent and require more investigation and template reading
}

// ClaimCreatePolicy is a type for values of spec.centralSpec.persistence.createClaim.
type ClaimCreatePolicy string

const (
	// ClaimCreate means a PVC should be created at install time.
	ClaimCreate ClaimCreatePolicy = "Create"
	// ClaimReuse means a pre-existing PVC should be used.
	ClaimReuse ClaimCreatePolicy = "Reuse"
)

// Exposure defines how central is exposed.
type Exposure struct {
	LoadBalancer *ExposureLoadBalancer `json:"loadBalancer,omitempty"`
	NodePort     *ExposureNodePort     `json:"nodePort,omitempty"`
	Route        *ExposureRoute        `json:"route,omitempty"`
}

// ExposureLoadBalancer defines settings for exposing central via a LoadBalancer.
type ExposureLoadBalancer struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Port    *int32  `json:"port,omitempty"`
	IP      *string `json:"ip,omitempty"`
}

// ExposureNodePort defines settings for exposing central via a NodePort.
type ExposureNodePort struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Port    *int32 `json:"port,omitempty"`
}

// ExposureRoute defines settings for exposing central via a Route.
type ExposureRoute struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// ScannerComponentSpec defines settings for the "scanner" component.
type ScannerComponentSpec struct {
	// Defaults to Enabled
	ScannerComponent *ScannerComponentPolicy `json:"scannerComponent,omitempty"`
	Replicas         *ScannerReplicas        `json:"replicas,omitempty"`
	Logging          *ScannerLogging         `json:"logging,omitempty"`
	Scanner          *common.DeploymentSpec  `json:"scanner,omitempty"`
	ScannerDB        *common.DeploymentSpec  `json:"scannerDB,omitempty"`
}

// ScannerComponentPolicy is a type for values of spec.scannerSpec.scannerComponent.
type ScannerComponentPolicy string

const (
	// ScannerComponentEnabled means that scanner should be installed.
	ScannerComponentEnabled ScannerComponentPolicy = "Enabled"
	// ScannerComponentDisabled means that scanner should not be installed.
	ScannerComponentDisabled ScannerComponentPolicy = "Disabled"
)

// ScannerReplicas defines replication settings of scanner.
type ScannerReplicas struct {
	AutoScaling *AutoScalingPolicy `json:"autoScaling,omitempty"`
	// Defaults to 3
	Replicas *int32 `json:"replicas,omitempty"`
	// Defaults to 2
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// Defaults to 5
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

// AutoScalingPolicy is a type for values of spec.scannerSpec.replicas.autoScaling.
type AutoScalingPolicy string

const (
	// ScannerAutoScalingEnabled means that scanner autoscaling should be enabled.
	ScannerAutoScalingEnabled AutoScalingPolicy = "Enabled"
	// ScannerAutoScalingDisabled means that scanner autoscaling should be disabled.
	ScannerAutoScalingDisabled AutoScalingPolicy = "Disabled"
)

// ScannerLogging defines logging settings for scanner.
type ScannerLogging struct {
	// Defaults to INFO.
	// TODO(ROX-7124): either document allowed values or drop the field
	Level *string `json:"level,omitempty"`
}

// -------------------------------------------------------------
// Status

// CentralStatus defines the observed state of Central.
type CentralStatus struct {
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []common.StackRoxCondition `json:"conditions"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	DeployedRelease *common.StackRoxRelease `json:"deployedRelease,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	CentralStatus *CentralComponentStatus `json:"centralStatus,omitempty"`
}

// CentralComponentStatus describes status specific to the central component.
type CentralComponentStatus struct {
	// If the admin password was auto-generated, it will be stored in this secret.
	// This field is omitted if the name of the secret is "central-admin-password".
	// See also spec.central.adminPasswordSecret
	GeneratedAdminPasswordSecret *corev1.LocalObjectReference `json:"generatedAdminPasswordSecret"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,central},{Deployment,v1,scanner},{Deployment,v1,scanner-db}}

// Central is the configuration template for the central services.
type Central struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CentralSpec   `json:"spec,omitempty"`
	Status CentralStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CentralList contains a list of Central
type CentralList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Central `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Central{}, &CentralList{})
}
