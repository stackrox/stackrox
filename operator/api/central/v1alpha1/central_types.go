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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// Important: Run "make generate manifests" to regenerate code after modifying this file

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

	// Implementation note: this is distinct from the secret that contains the htpasswd-encoded password mounted in central.
	// TODO(ROX-7242): expose the secret name unconditionally

	// A Kubernetes secret that contains a TLS certificate and key for HTTPS serving of the web UI.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="User-facing TLS secret"
	DefaultTLSSecret *common.LocalSecretReference `json:"defaultTLSSecret,omitempty"`

	// Reference to a user-created Secret with admin password stored in data item "value".
	// If omitted, the operator will instead auto-generate a password, create such secret and
	// expose the name of that secret (if it is different from the name "central-admin-password") in status.central.generatedAdminPasswordSecret
	AdminPasswordSecret *common.LocalSecretReference `json:"adminPasswordSecret,omitempty"`
	Persistence         *Persistence                 `json:"persistence,omitempty"`
	Exposure            *Exposure                    `json:"exposure,omitempty"`
	// TODO(ROX-7123): determine whether we want to make `extraMounts` available in the operator
	// TODO(ROX-7112): should we expose central.config? It's exposed in helm charts but not documented in help.stackrox.com.
	// TODO(ROX-7147): design central endpoint
}

// GetHostPath returns Central's configured host path
func (c *CentralComponentSpec) GetHostPath() string {
	if c == nil {
		return ""
	}
	if c.Persistence == nil {
		return ""
	}
	if c.Persistence.HostPath == nil {
		return ""
	}

	return pointer.StringPtrDerefOr(c.Persistence.HostPath.Path, "")
}

// GetAdminPasswordSecret provides a way to retrieve the admin password that is safe to use on a nil receiver object.
func (c *CentralComponentSpec) GetAdminPasswordSecret() *common.LocalSecretReference {
	if c == nil {
		return nil
	}
	return c.AdminPasswordSecret
}

// Persistence defines persistence settings for central.
type Persistence struct {
	HostPath              *HostPathSpec          `json:"hostPath,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`
}

// HostPathSpec defines settings for host path config.
type HostPathSpec struct {
	Path *string `json:"path,omitempty"`
}

// PersistentVolumeClaim defines PVC-based persistence settings.
type PersistentVolumeClaim struct {
	ClaimName        *string           `json:"claimName,omitempty"`
	StorageClassName *string           `json:"storageClassName,omitempty"`
	Size             resource.Quantity `json:"size,omitempty"`
}

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
	ScannerComponent *ScannerComponentPolicy   `json:"scannerComponent,omitempty"`
	Analyzer         *ScannerAnalyzerComponent `json:"analyzer,omitempty"`
	DB               *common.DeploymentSpec    `json:"db,omitempty"`
}

// GetAnalyzer returns the analyzer component even if receiver is nil
func (s *ScannerComponentSpec) GetAnalyzer() *ScannerAnalyzerComponent {
	if s == nil {
		return nil
	}
	return s.Analyzer
}

// IsEnabled checks whether scanner is enabled. This method is safe to be used with nil receivers.
func (s *ScannerComponentSpec) IsEnabled() bool {
	if s == nil || s.ScannerComponent == nil {
		return true // enabled by default
	}
	return *s.ScannerComponent == ScannerComponentEnabled
}

// ScannerComponentPolicy is a type for values of spec.scannerSpec.scannerComponent.
type ScannerComponentPolicy string

const (
	// ScannerComponentEnabled means that scanner should be installed.
	ScannerComponentEnabled ScannerComponentPolicy = "Enabled"
	// ScannerComponentDisabled means that scanner should not be installed.
	ScannerComponentDisabled ScannerComponentPolicy = "Disabled"
)

// ScannerAnalyzerComponent describes the analyzer component
type ScannerAnalyzerComponent struct {
	common.DeploymentSpec `json:",inline"`
	Scaling               *ScannerAnalyzerScaling `json:"scaling,omitempty"`
}

// GetScaling returns scaling config even if receiver is nil
func (s *ScannerAnalyzerComponent) GetScaling() *ScannerAnalyzerScaling {
	if s == nil {
		return nil
	}
	return s.Scaling
}

// ScannerAnalyzerScaling defines replication settings of the analyzer.
type ScannerAnalyzerScaling struct {
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

// AdminPasswordStatus shows status related to the admin password.
type AdminPasswordStatus struct {
	// Info stores information on how to obtain the admin password.
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Info string `json:"info,omitempty"`
}

// CentralComponentStatus describes status specific to the central component.
type CentralComponentStatus struct {
	// AdminPassword stores information related to the auto-generated admin password.
	AdminPassword *AdminPasswordStatus `json:"adminPassword,omitempty"`
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

var (
	// CentralGVK is the GVK for the Central type.
	CentralGVK = GroupVersion.WithKind("Central")
)
