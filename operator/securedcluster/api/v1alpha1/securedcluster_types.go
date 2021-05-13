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
	common "github.com/stackrox/rox/operator/common/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make generate manifests" to regenerate code after modifying this file
// TODO(ROX-7110): prevent merging PRs if manifests are not up to date.

// -------------------------------------------------------------
// Spec

// SecuredClusterSpec defines the desired state of SecuredCluster
type SecuredClusterSpec struct {
	// TODO(ROX-7125): decide how to guarantee immutability; use metadata.name instead?
	ClusterName string `json:"clusterName"`
	// Address of the Central endpoint, including the port number.
	// If using a non-gRPC capable LoadBalancer, use the WebSocket protocol by prefixing
	// the endpoint address with wss://.
	CentralEndpoint string `json:"centralEndpoint"`

	TLS              *common.TLSConfig              `json:"tls,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference  `json:"imagePullSecrets,omitempty"`
	Sensor           *SensorComponentSpec           `json:"sensor,omitempty"`
	AdmissionControl *AdmissionControlComponentSpec `json:"admissionControl,omitempty"`
	Collector        *CollectorComponentSpec        `json:"collector,omitempty"`
	// Customizations to apply on all secured cluster components.
	Customize *common.CustomizeSpec `json:"customize,omitempty"`
	// TODO(ROX-7150): We do not support setting image in the CRs because they are determined by
	// the operator version whose lifecycle is orthogonal to that of the CR.
}

// SensorComponentSpec defines settings for sensor.
type SensorComponentSpec struct {
	ContainerSpec `json:",inline"`
	NodeSelector  map[string]string `json:"nodeSelector,omitempty"`
	// Secret that contains cert and key. Omit means: autogenerate.
	ServiceTLS *corev1.LocalObjectReference `json:"serviceTLS,omitempty"`

	// Address of the Sensor endpoint including port number. No trailing slash.
	// Rarely needs to be changed.
	Endpoint *string `json:"endpoint,omitempty"`
}

// AdmissionControlComponentSpec defines settings for admission control.
type AdmissionControlComponentSpec struct {
	ContainerSpec `json:",inline"`
	// Secret that contains cert and key. Omit means: autogenerate.
	ServiceTLS *corev1.LocalObjectReference `json:"serviceTLS,omitempty"`

	// This setting controls whether the cluster is configured to contact the StackRox
	// Kubernetes Security Platform with `AdmissionReview` requests for create events on
	// Kubernetes objects.
	ListenOnCreates *bool `json:"listenOnCreates,omitempty"`

	// This setting controls whether the cluster is configured to contact the StackRox Kubernetes
	// Security Platform with `AdmissionReview` requests for update events on Kubernetes objects.
	ListenOnUpdates *bool `json:"listenOnUpdates,omitempty"`

	// This setting controls whether the cluster is configured to contact the StackRox
	// Kubernetes Security Platform with `AdmissionReview` requests for update Kubernetes events
	// like exec and portforward.
	// Defaults to `false` on OpenShift, to `true` otherwise.
	ListenOnEvents *bool `json:"listenOnEvents,omitempty"`
}

// CollectorComponentSpec defines settings for collector.
type CollectorComponentSpec struct {
	Collection      *CollectionMethod      `json:"collection,omitempty"`
	TaintToleration *TaintTolerationPolicy `json:"taintToleration,omitempty"`

	Collector  *CollectorContainerSpec `json:"collector,omitempty"`
	Compliance *ContainerSpec          `json:"compliance,omitempty"`
	// Secret that contains cert and key. Omit means: autogenerate.
	ServiceTLS *corev1.LocalObjectReference `json:"serviceTLS,omitempty"`
	// Customizations to apply on the collector DaemonSet.
	Customize *common.CustomizeSpec `json:"customize,omitempty"`
}

// CollectionMethod is a type for values of spec.collector.collection
type CollectionMethod string

const (
	// CollectionEBPF means: use EBPF collection.
	CollectionEBPF = "EBPF"
	// CollectionModule means: use KERNEL_MODULE collection.
	CollectionModule = "KernelModule"
	// CollectionNone means: NO_COLLECTION.
	CollectionNone = "None"
)

// TaintTolerationPolicy is a type for values of spec.collector.taintToleration
type TaintTolerationPolicy string

const (
	// TaintTolerate means tolerations are applied to collector, and the collector pods can schedule onto all nodes with taints.
	TaintTolerate TaintTolerationPolicy = "TolerateTaints"
	// TaintAvoid means no tolerations are applied, and the collector pods won't schedule onto nodes with taints.
	TaintAvoid TaintTolerationPolicy = "AvoidTaints"
)

// CollectorContainerSpec defines settings for the collector container.
type CollectorContainerSpec struct {
	ContainerSpec `json:",inline"`
	ImageFlavor   *CollectorImageFlavor `json:"imageFlavor,omitempty"`
}

// ContainerSpec defines settings common to secured cluster components.
type ContainerSpec struct {
	Resources       *common.Resources  `json:"resources,omitempty"`
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Customizations to apply on this container.
	Customize *ContainerCustomizeSpec `json:"customize,omitempty"`
}

// ContainerCustomizeSpec contains customizations to apply on a container.
type ContainerCustomizeSpec struct {
	// Custom environment variables to set on this container.
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

// -------------------------------------------------------------
// Status

// SecuredClusterStatus defines the observed state of SecuredCluster
type SecuredClusterStatus struct {
	Conditions      []common.StackRoxCondition `json:"conditions"`
	DeployedRelease *common.StackRoxRelease    `json:"deployedRelease,omitempty"`
	SensorStatus    *SensorComponentStatus     `json:"sensorStatus,omitempty"`
}

// SensorComponentStatus describes status specific to the sensor component.
type SensorComponentStatus struct {
	ClusterID *string `json:"clusterID,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SecuredCluster is the Schema for the securedclusters API
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
