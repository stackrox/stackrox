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
	// The unique name of this cluster, as it will be shown in the Red Hat Advanced Cluster Security UI.
	// Note: Once a name is set here, you will not be able to change it again. You will need to delete
	// and re-create this object in order to register a cluster with a new name.
	//+kubebuilder:validation:Required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	ClusterName string `json:"clusterName"`

	// The endpoint of the Red Hat Advanced Cluster Security Central instance to connect to,
	// including the port number. If using a non-gRPC capable load balancer, use the WebSocket protocol by
	// prefixing the endpoint address with wss://.
	// Note: when leaving this blank, Sensor will attempt to connect to a Central instance running in the same
	// namespace.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	CentralEndpoint string `json:"centralEndpoint,omitempty"`

	// Settings for the Sensor component.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="Sensor Settings"
	Sensor *SensorComponentSpec `json:"sensor,omitempty"`

	// Settings for the Admission Control component, which is necessary for preventive policy enforcement,
	// and for Kubernetes event monitoring.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4,displayName="Admission Control Settings"
	AdmissionControl *AdmissionControlComponentSpec `json:"admissionControl,omitempty"`

	// Settings for the components running on each node in the cluster (Collector and Compliance).
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=5,displayName="Per Node Settings"
	PerNode *PerNodeSpec `json:"perNode,omitempty"`

	// Allows you to specify additional trusted Root CAs.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=6
	TLS *common.TLSConfig `json:"tls,omitempty"`

	// Additional image pull secrets to be taken into account for pulling images.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image Pull Secrets",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullSecrets []common.LocalSecretReference `json:"imagePullSecrets,omitempty"`

	// Customizations to apply on all Central Services components.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Customizations,order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Customize *common.CustomizeSpec `json:"customize,omitempty"`

	// Miscellaneous settings.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Miscellaneous,order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Misc *common.MiscSpec `json:"misc,omitempty"`
}

// SensorComponentSpec defines settings for sensor.
type SensorComponentSpec struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	common.DeploymentSpec `json:",inline"`
}

// AdmissionControlComponentSpec defines settings for the admission controller configuration.
type AdmissionControlComponentSpec struct {
	// Set this to 'true' to enable preventive policy enforcement for object creations.
	//+kubebuilder:validation:Default=false
	//+kubebuilder:default=false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	ListenOnCreates *bool `json:"listenOnCreates,omitempty"`

	// Set this to 'true' to enable preventive policy enforcement for object updates.
	//
	// Note: this will not have any effect unless 'Listen On Creates' is set to 'true' as well.
	//+kubebuilder:validation:Default=false
	//+kubebuilder:default=false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	ListenOnUpdates *bool `json:"listenOnUpdates,omitempty"`

	// Set this to 'true' to enable monitoring and enforcement for Kubernetes events (port-forward and exec).
	//+kubebuilder:validation:Default=true
	//+kubebuilder:default=true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	ListenOnEvents *bool `json:"listenOnEvents,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	common.DeploymentSpec `json:",inline"`
}

// PerNodeSpec declares configuration settings for components which are deployed to all nodes.
type PerNodeSpec struct {
	// Settings for the Collector container, which is responsible for collecting process and networking
	// activity at the host level.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Collector Settings"
	Collector *CollectorContainerSpec `json:"collector,omitempty"`

	// Settings for the Compliance container, which is responsible for checking host-level configurations.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,displayName="Compliance Settings"
	Compliance *ContainerSpec `json:"compliance,omitempty"`

	// To ensure comprehensive monitoring of your cluster activity, Red Hat Advanced Cluster Security
	// will run services on every node in the cluster, including tainted nodes by default. If you do
	// not want this behavior, please select 'AvoidTaints' here.
	//+kubebuilder:validation:Default=TolerateTaints
	//+kubebuilder:default=TolerateTaints
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	TaintToleration *TaintTolerationPolicy `json:"taintToleration,omitempty"`
}

// CollectionMethod defines the method of collection used by collector. Options are 'EBPF', 'KernelModule' or 'None'.
//+kubebuilder:validation:Enum=EBPF;KernelModule;NoCollection
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
//+kubebuilder:validation:Enum=TolerateTaints;AvoidTaints
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
	// The method for system-level data collection. Kernel module is recommended.
	// If you select "NoCollection", you will not be able to see any information about network activity
	// and process executions. The remaining settings in these section will not have any effect.
	//+kubebuilder:validation:Default=KernelModule
	//+kubebuilder:default=KernelModule
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Collection *CollectionMethod `json:"collection,omitempty"`

	// The image flavor to use for collector. "Regular" images are bigger in size, but contain kernel modules
	// for most kernels. If you use the "Slim" image flavor, you must ensure that your Central instance
	// is connected to the internet, or regularly receives Collector Support Package updates (for further
	// instructions, please refer to the documentation).
	//+kubebuilder:validation:Default=Regular
	//+kubebuilder:default=Regular
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	ImageFlavor *CollectorImageFlavor `json:"imageFlavor,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	ContainerSpec `json:",inline"`
}

// ContainerSpec defines container settings.
type ContainerSpec struct {
	// Allows overriding the default resource settings for this component. Please consult the documentation
	// for an overview of default resource requirements and a sizing guide.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"},order=100
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// CollectorImageFlavor is a type for values of spec.collector.collector.imageFlavor
//+kubebuilder:validation:Enum=Regular;Slim
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
	Conditions      []common.StackRoxCondition `json:"conditions"`
	DeployedRelease *common.StackRoxRelease    `json:"deployedRelease,omitempty"`

	// The deployed version of the product.
	//+operator-sdk:csv:customresourcedefinitions:type=status,order=1
	ProductVersion string `json:"productVersion,omitempty"`

	// The assigned cluster name per the spec. This cannot be changed afterwards. If you need to change the
	// cluster name, please delete and recreate this resource.
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Cluster Name",order=2
	ClusterName string `json:"clusterName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,admission-control},{DaemonSet,v1,collector},{Deployment,v1,sensor}}

// SecuredCluster is the configuration template for the secured cluster services. These include Sensor, which is
// responsible for the connection to Central, and Collector, which performs host-level collection of process and
// network events.<p>
// **Important:** Please see the _Installation Prerequisites_ on the main page before deploying, or consult the RHACS
// documentation on creating cluster init bundles.
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

var (
	// SecuredClusterGVK is the GVK for the SecuredCluster type.
	SecuredClusterGVK = GroupVersion.WithKind("SecuredCluster")
)
