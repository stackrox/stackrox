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
	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make generate manifests" to regenerate code after modifying this file

// -------------------------------------------------------------
// Spec

// SecuredClusterSpec defines the desired configuration state of a secured cluster.
type SecuredClusterSpec struct {
	// TODO(ROX-7125): decide how to guarantee immutability; use metadata.name instead?

	// ClusterName should specify the name assigned to your secured cluster.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	ClusterName string `json:"clusterName"`
	// CentralEndpoint should specify the address of the Central endpoint, including the port number.
	// If using a non-gRPC capable LoadBalancer, use the WebSocket protocol by prefixing the endpoint address
	// with wss://.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	CentralEndpoint *string `json:"centralEndpoint,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec
	TLS *TLSConfig `json:"tls,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Sensor *SensorComponentSpec `json:"sensor,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	AdmissionControl *AdmissionControlComponentSpec `json:"admissionControl,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	PerNode *PerNodeSpec `json:"perNode,omitempty"`
	// Customizations to apply on all secured cluster components.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Customize *common.CustomizeSpec `json:"customize,omitempty"`
}

// SensorComponentSpec defines settings for sensor.
type SensorComponentSpec struct {
	ContainerSpec `json:",inline"`

	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// TLSConfig defines TLS-related settings
type TLSConfig struct {
	AdditionalCAs []common.AdditionalCA `json:"additionalCAs,omitempty"`
}

// AdmissionControlComponentSpec defines settings for the admission controller configuration.
type AdmissionControlComponentSpec struct {
	ContainerSpec `json:",inline"`

	// ListenOnCreates controls whether Kubernetes is configured to contact Secured Cluster Services with
	// `AdmissionReview` requests for workload creation events.
	ListenOnCreates *bool `json:"listenOnCreates,omitempty"`

	// ListenOnUpdates controls whether Kubernetes is configured to contact Secured Cluster Services with
	// `AdmissionReview` requests for update events on Kubernetes objects.
	ListenOnUpdates *bool `json:"listenOnUpdates,omitempty"`

	// ListenOnEvents controls whether Kubernetes is configured to contact Secured Cluster Services with
	// `AdmissionReview` requests for update Kubernetes events like exec and portforward.
	// Defaults to `false` on OpenShift, to `true` otherwise.
	ListenOnEvents *bool `json:"listenOnEvents,omitempty"`
}

// PerNodeSpec declares configuration settings for components which are deployed to all nodes.
type PerNodeSpec struct {
	Collection      *CollectionMethod      `json:"collection,omitempty"`
	TaintToleration *TaintTolerationPolicy `json:"taintToleration,omitempty"`

	Collector  *CollectorContainerSpec `json:"collector,omitempty"`
	Compliance *ContainerSpec          `json:"compliance,omitempty"`
}

// CollectionMethod defines the method of collection used by collector. Options are 'EBPF', 'KernelModule' or 'None'.
type CollectionMethod string

const (
	// CollectionEBPF means: use EBPF collection.
	CollectionEBPF CollectionMethod = "EBPF"
	// CollectionKernelModule means: use KERNEL_MODULE collection.
	CollectionKernelModule CollectionMethod = "KernelModule"
	// CollectionNone means: NO_COLLECTION.
	CollectionNone CollectionMethod = "NoCollection"
)

// Pointer returns the given CollectionMethod as a pointer, needed in k8s resource structs.
func (c CollectionMethod) Pointer() *CollectionMethod {
	return &c
}

// TaintTolerationPolicy is a type for values of spec.collector.taintToleration
type TaintTolerationPolicy string

const (
	// TaintTolerate means tolerations are applied to collector, and the collector pods can schedule onto all nodes with taints.
	TaintTolerate TaintTolerationPolicy = "TolerateTaints"
	// TaintAvoid means no tolerations are applied, and the collector pods won't schedule onto nodes with taints.
	TaintAvoid TaintTolerationPolicy = "AvoidTaints"
)

// Pointer returns the given TaintTolerationPolicy as a pointer, needed in k8s resource structs.
func (t TaintTolerationPolicy) Pointer() *TaintTolerationPolicy {
	return &t
}

// CollectorContainerSpec defines settings for the collector container.
type CollectorContainerSpec struct {
	ContainerSpec `json:",inline"`
	ImageFlavor   *CollectorImageFlavor `json:"imageFlavor,omitempty"`
}

// ContainerSpec defines container settings.
type ContainerSpec struct {
	Resources *common.Resources `json:"resources,omitempty"`
	// Customize specifies additional attributes for all containers.
	Customize *ContainerCustomizeSpec `json:"customize,omitempty"`
}

// ContainerCustomizeSpec contains customizations to apply on a container.
type ContainerCustomizeSpec struct {
	// EnvVars set custom environment variables on pods' containers.
	EnvVars map[string]string `json:"envVars,omitempty"`
}

// CollectorImageFlavor is a type for values of spec.collector.collector.imageFlavor
type CollectorImageFlavor string

const (
	// ImageFlavorRegular means to use regular collector images.
	ImageFlavorRegular CollectorImageFlavor = "Regular"
	// ImageFlavorSlim means to use slim collector images.
	ImageFlavorSlim CollectorImageFlavor = "Slim"
)

// Pointer returns the given CollectorImageFlavor as a pointer, needed in k8s resource structs.
func (c CollectorImageFlavor) Pointer() *CollectorImageFlavor {
	return &c
}

// -------------------------------------------------------------
// Status

// SecuredClusterStatus defines the observed state of SecuredCluster
type SecuredClusterStatus struct {
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []common.StackRoxCondition `json:"conditions"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	DeployedRelease *common.StackRoxRelease `json:"deployedRelease,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	SensorStatus *SensorComponentStatus `json:"sensorStatus,omitempty"`
}

// SensorComponentStatus describes status specific to the sensor component.
type SensorComponentStatus struct {
	ClusterID *string `json:"clusterID,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,admission-control},{DaemonSet,v1,collector},{Deployment,v1,sensor}}

// SecuredCluster is the configuration template for the secured cluster services.
type SecuredCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecuredClusterSpec   `json:"spec,omitempty"`
	Status SecuredClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SecuredClusterList contains a list of SecuredCluster
type SecuredClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecuredCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecuredCluster{}, &SecuredClusterList{})
}
