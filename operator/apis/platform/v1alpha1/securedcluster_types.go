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

	// Custom labels associated with a secured cluster in Red Hat Advanced Cluster Security.
	ClusterLabels map[string]string `json:"clusterLabels,omitempty"`

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

	// Settings relating to the ingestion of Kubernetes audit logs.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=6,displayName="Kubernetes Audit Logs Ingestion Settings"
	AuditLogs *AuditLogsSpec `json:"auditLogs,omitempty"`

	// Settings for the Scanner component, which is responsible for vulnerability scanning of container
	// images stored in a cluster-local image repository.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=7,displayName="Scanner Component Settings"
	Scanner *LocalScannerComponentSpec `json:"scanner,omitempty"`

	// Allows you to specify additional trusted Root CAs.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=8
	TLS *TLSConfig `json:"tls,omitempty"`

	// Additional image pull secrets to be taken into account for pulling images.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image Pull Secrets",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullSecrets []LocalSecretReference `json:"imagePullSecrets,omitempty"`

	// Customizations to apply on all Central Services components.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Customizations,order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Customize *CustomizeSpec `json:"customize,omitempty"`

	// Overlays
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Overlays,order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Overlays []*K8sObjectOverlay `json:"overlays,omitempty"`

	// Monitoring configuration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=13,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Monitoring *GlobalMonitoring `json:"monitoring,omitempty"`

	// Set this parameter to override the default registry in images. For example, nginx:latest -> <registry override>/library/nginx:latest
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom Default Image Registry",order=14,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RegistryOverride string `json:"registryOverride,omitempty"`
}

// SensorComponentSpec defines settings for sensor.
type SensorComponentSpec struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	DeploymentSpec `json:",inline"`
}

// AdmissionControlComponentSpec defines settings for the admission controller configuration.
type AdmissionControlComponentSpec struct {
	// Set this to 'true' to enable preventive policy enforcement for object creations.
	//+kubebuilder:default=true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	ListenOnCreates *bool `json:"listenOnCreates,omitempty"`

	// Set this to 'true' to enable preventive policy enforcement for object updates.
	//
	// Note: this will not have any effect unless 'Listen On Creates' is set to 'true' as well.
	//+kubebuilder:default=true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	ListenOnUpdates *bool `json:"listenOnUpdates,omitempty"`

	// Set this to 'true' to enable monitoring and enforcement for Kubernetes events (port-forward and exec).
	//+kubebuilder:default=true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	ListenOnEvents *bool `json:"listenOnEvents,omitempty"`

	// Should inline scanning be performed on previously unscanned images during a deployments admission review.
	//+kubebuilder:default=DoNotScanInline
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	ContactImageScanners *ImageScanPolicy `json:"contactImageScanners,omitempty"`

	// Maximum timeout period for admission review, upon which admission review will fail open.
	// Use it to set request timeouts when you enable inline image scanning.
	// The default kubectl timeout is 30 seconds; taking padding into account, this should not exceed 25 seconds.
	//+kubebuilder:default=20
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=25
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=5
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// Enables teams to bypass admission control in a monitored manner in the event of an emergency.
	//+kubebuilder:default=BreakGlassAnnotation
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=6
	Bypass *BypassPolicy `json:"bypass,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=7
	DeploymentSpec `json:",inline"`

	// The number of replicas of the admission control pod.
	//+kubebuilder:default=3
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=8
	Replicas *int32 `json:"replicas,omitempty"`
}

// ImageScanPolicy defines whether images should be scanned at admission control time.
// +kubebuilder:validation:Enum=ScanIfMissing;DoNotScanInline
type ImageScanPolicy string

const (
	// ScanIfMissing means that images which do not have a known scan result should be scanned in scope of an admission request.
	ScanIfMissing ImageScanPolicy = "ScanIfMissing"
	// DoNotScanInline means that images which do not have a known scan result will not be scanned when processing an admission request.
	DoNotScanInline ImageScanPolicy = "DoNotScanInline"
)

// Pointer returns the given ImageScanPolicy as a pointer, needed in k8s resource structs.
func (p ImageScanPolicy) Pointer() *ImageScanPolicy {
	return &p
}

// BypassPolicy defines whether admission controller can be bypassed.
// +kubebuilder:validation:Enum=BreakGlassAnnotation;Disabled
type BypassPolicy string

const (
	// BypassBreakGlassAnnotation means that admission controller can be bypassed by adding an admission.stackrox.io/break-glass annotation to a resource.
	// Bypassing the admission controller triggers a policy violation which includes deployment details.
	// We recommend providing an issue-tracker link or some other reference as the value of this annotation so that others can understand why you bypassed the admission controller.
	BypassBreakGlassAnnotation BypassPolicy = "BreakGlassAnnotation"
	// BypassDisabled means that admission controller cannot be bypassed.
	BypassDisabled BypassPolicy = "Disabled"
)

// Pointer returns the given BypassPolicy as a pointer, needed in k8s resource structs.
func (p BypassPolicy) Pointer() *BypassPolicy {
	return &p
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

	// Settings for the Node-Inventory container, which is responsible for scanning the Nodes' filesystem.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="Node Scanning Settings"
	NodeInventory *ContainerSpec `json:"nodeInventory,omitempty"`

	// To ensure comprehensive monitoring of your cluster activity, Red Hat Advanced Cluster Security
	// will run services on every node in the cluster, including tainted nodes by default. If you do
	// not want this behavior, please select 'AvoidTaints' here.
	//+kubebuilder:default=TolerateTaints
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	TaintToleration *TaintTolerationPolicy `json:"taintToleration,omitempty"`
}

// CollectionMethod defines the method of collection used by collector. Options are 'EBPF', 'CORE_BPF', 'None', or 'KernelModule'. Note that the collection method will be switched to EBPF if KernelModule is used.
// +kubebuilder:validation:Enum=EBPF;CORE_BPF;NoCollection;KernelModule
type CollectionMethod string

const (
	// CollectionEBPF means: use EBPF collection.
	CollectionEBPF CollectionMethod = "EBPF"
	// CollectionCOREBPF means: use CORE_BPF collection.
	CollectionCOREBPF CollectionMethod = "CORE_BPF"
	// CollectionNone means: NO_COLLECTION.
	CollectionNone CollectionMethod = "NoCollection"
	// CollectionKernelModule means: use KERNEL_MODULE collection.
	CollectionKernelModule CollectionMethod = "KernelModule"
)

// Pointer returns the given CollectionMethod as a pointer, needed in k8s resource structs.
func (c CollectionMethod) Pointer() *CollectionMethod {
	return &c
}

// AuditLogsSpec configures settings related to audit log ingestion.
type AuditLogsSpec struct {
	// Whether collection of Kubernetes audit logs should be enabled or disabled. Currently, this is only
	// supported on OpenShift 4, and trying to enable it on non-OpenShift 4 clusters will result in an error.
	// Use the 'Auto' setting to enable it on compatible environments, and disable it elsewhere.
	//+kubebuilder:default=Auto
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Collection *AuditLogsCollectionSetting `json:"collection,omitempty"`
}

// AuditLogsCollectionSetting determines if audit log collection is enabled.
// +kubebuilder:validation:Enum=Auto;Disabled;Enabled
type AuditLogsCollectionSetting string

const (
	// AuditLogsCollectionAuto means to configure audit logs collection according to the environment (enable on
	// OpenShift 4.x, disable on all other environments).
	AuditLogsCollectionAuto AuditLogsCollectionSetting = "Auto"
	// AuditLogsCollectionDisabled means to disable audit logs collection.
	AuditLogsCollectionDisabled AuditLogsCollectionSetting = "Disabled"
	// AuditLogsCollectionEnabled means to enable audit logs collection.
	AuditLogsCollectionEnabled AuditLogsCollectionSetting = "Enabled"
)

// Pointer returns a pointer with the given value.
func (s AuditLogsCollectionSetting) Pointer() *AuditLogsCollectionSetting {
	return &s
}

// TaintTolerationPolicy is a type for values of spec.collector.taintToleration
// +kubebuilder:validation:Enum=TolerateTaints;AvoidTaints
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
	// The method for system-level data collection. EBPF is recommended.
	// If you select "NoCollection", you will not be able to see any information about network activity
	// and process executions. The remaining settings in these section will not have any effect.
	//+kubebuilder:default=EBPF
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:EBPF", "urn:alm:descriptor:com.tectonic.ui:select:CORE_BPF", "urn:alm:descriptor:com.tectonic.ui:select:NoCollection"}
	Collection *CollectionMethod `json:"collection,omitempty"`

	// The image flavor to use for collector. "Regular" images are bigger in size, but contain probes
	// for most kernels. If you use the "Slim" image flavor, you must ensure that your Central instance
	// is connected to the internet, or regularly receives Collector Support Package updates (for further
	// instructions, please refer to the documentation).
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
// +kubebuilder:validation:Enum=Regular;Slim
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

// Note the following struct should mostly match ScannerComponentSpec for the Central's type. Different Scanner
// types struct are maintained because of UI exposed documentation differences.

// LocalScannerComponentSpec defines settings for the "scanner" component.
type LocalScannerComponentSpec struct {
	// If you do not want to deploy the Red Hat Advanced Cluster Security Scanner, you can disable it here
	// (not recommended).
	// If you do so, all the settings in this section will have no effect.
	//+kubebuilder:default=AutoSense
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Scanner Component",order=1
	ScannerComponent *LocalScannerComponentPolicy `json:"scannerComponent,omitempty"`

	// Settings pertaining to the analyzer deployment, such as for autoscaling.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Analyzer *ScannerAnalyzerComponent `json:"analyzer,omitempty"`

	// Settings pertaining to the database used by the Red Hat Advanced Cluster Security Scanner.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="DB"
	DB *DeploymentSpec `json:"db,omitempty"`
}

// LocalScannerComponentPolicy is a type for values of spec.scanner.scannerComponent.
// +kubebuilder:validation:Enum=AutoSense;Disabled
type LocalScannerComponentPolicy string

const (
	// LocalScannerComponentAutoSense means that scanner should be installed,
	// unless there is a Central resource in the same namespace.
	// In that case typically a central scanner will be deployed as a component of Central.
	LocalScannerComponentAutoSense LocalScannerComponentPolicy = "AutoSense"
	// LocalScannerComponentDisabled means that scanner should not be installed.
	LocalScannerComponentDisabled LocalScannerComponentPolicy = "Disabled"
)

// Pointer returns the pointer of the policy.
func (l LocalScannerComponentPolicy) Pointer() *LocalScannerComponentPolicy {
	return &l
}

// -------------------------------------------------------------
// Status

// SecuredClusterStatus defines the observed state of SecuredCluster
type SecuredClusterStatus struct {
	Conditions      []StackRoxCondition `json:"conditions"`
	DeployedRelease *StackRoxRelease    `json:"deployedRelease,omitempty"`

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
//+operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,""},{DaemonSet,v1,""}}
//+genclient

// SecuredCluster is the configuration template for the secured cluster services. These include Sensor, which is
// responsible for the connection to Central, and Collector, which performs host-level collection of process and
// network events.<p>
// **Important:** Please see the _Installation Prerequisites_ on the main RHACS operator page before deploying, or
// consult the RHACS documentation on creating cluster init bundles.
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

var (
	// SecuredClusterGVK is the GVK for the SecuredCluster type.
	SecuredClusterGVK = SchemeGroupVersion.WithKind("SecuredCluster")
)
