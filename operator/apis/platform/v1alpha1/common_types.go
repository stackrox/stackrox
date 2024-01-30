//+kubebuilder:object:generate=true

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// MiscSpec defines miscellaneous settings for custom resources.
type MiscSpec struct {
	// Deprecated field. This field will be removed in a future release.
	// Set this to true to have the operator create SecurityContextConstraints (SCCs) for the operands. This
	// isn't usually needed, and may interfere with other workloads.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Create SecurityContextConstraints for Operand",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	CreateSCCs *bool `json:"createSCCs,omitempty"`
}

// CustomizeSpec defines customizations to apply.
type CustomizeSpec struct {
	// Custom labels to set on all managed objects.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Labels map[string]string `json:"labels,omitempty"`
	// Custom annotations to set on all managed objects.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Annotations map[string]string `json:"annotations,omitempty"`

	// Custom environment variables to set on managed pods' containers.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="Environment Variables"
	EnvVars []corev1.EnvVar `json:"envVars,omitempty"`
}

// DeploymentSpec defines settings that affect a deployment.
type DeploymentSpec struct {
	// Allows overriding the default resource settings for this component. Please consult the documentation
	// for an overview of default resource requirements and a sizing guide.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"},order=100
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// If you want this component to only run on specific nodes, you can configure a node selector here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=101
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If you want this component to only run on specific nodes, you can configure tolerations of tainted nodes.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:tolerations"},order=102
	Tolerations []*corev1.Toleration `json:"tolerations,omitempty"`
}

// StackRoxCondition defines a condition for a StackRox custom resource.
type StackRoxCondition struct {
	Type    ConditionType   `json:"type"`
	Status  ConditionStatus `json:"status"`
	Reason  ConditionReason `json:"reason,omitempty"`
	Message string          `json:"message,omitempty"`

	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// ConditionType is a type of values of condition type.
type ConditionType string

// ConditionStatus is a type of values of condition status.
type ConditionStatus string

// ConditionReason is a type of values of condition reason.
type ConditionReason string

// These are the allowed values for StackRoxCondition fields.
const (
	ConditionInitialized    ConditionType = "Initialized"
	ConditionDeployed       ConditionType = "Deployed"
	ConditionReleaseFailed  ConditionType = "ReleaseFailed"
	ConditionIrreconcilable ConditionType = "Irreconcilable"

	StatusTrue    ConditionStatus = "True"
	StatusFalse   ConditionStatus = "False"
	StatusUnknown ConditionStatus = "Unknown"

	ReasonInstallSuccessful   ConditionReason = "InstallSuccessful"
	ReasonUpgradeSuccessful   ConditionReason = "UpgradeSuccessful"
	ReasonUninstallSuccessful ConditionReason = "UninstallSuccessful"
	ReasonInstallError        ConditionReason = "InstallError"
	ReasonUpgradeError        ConditionReason = "UpgradeError"
	ReasonReconcileError      ConditionReason = "ReconcileError"
	ReasonUninstallError      ConditionReason = "UninstallError"
)

// StackRoxRelease describes the Helm "release" that was most recently applied.
type StackRoxRelease struct {
	Version string `json:"version,omitempty"`
}

// AdditionalCA defines a certificate for an additional Certificate Authority.
type AdditionalCA struct {
	// Must be a valid file basename
	Name string `json:"name"`
	// PEM format
	Content string `json:"content"`
}

// TLSConfig defines common TLS-related settings for all components.
type TLSConfig struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Additional CAs"
	AdditionalCAs []AdditionalCA `json:"additionalCAs,omitempty"`
}

// LocalSecretReference is a reference to a secret within the same namespace.
type LocalSecretReference struct {
	// The name of the referenced secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	Name string `json:"name"`
}

// LocalConfigMapReference is a reference to a config map within the same namespace.
type LocalConfigMapReference struct {
	// The name of the referenced config map.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:ConfigMap"}
	Name string `json:"name"`
}

// ScannerAnalyzerComponent describes the analyzer component
type ScannerAnalyzerComponent struct {
	// Controls the number of analyzer replicas and autoscaling.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Scaling *ScannerComponentScaling `json:"scaling,omitempty"`

	DeploymentSpec `json:",inline"`
}

// GetScaling returns scaling config even if receiver is nil
func (s *ScannerAnalyzerComponent) GetScaling() *ScannerComponentScaling {
	if s == nil {
		return nil
	}
	return s.Scaling
}

// ScannerV4Component defines common configuration for Scanner V4 indexer and matcher components.
type ScannerV4Component struct {
	// Controls the number of replicas and autoscaling for this component.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Scaling        *ScannerComponentScaling `json:"scaling,omitempty"`
	DeploymentSpec `json:",inline"`
}

// ScannerV4DB defines configuration for the Scanner V4 database component.
type ScannerV4DB struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Persistence    *ScannerV4Persistence `json:"persistence,omitempty"`
	DeploymentSpec `json:",inline"`
}

// ScannerV4Persistence defines persistence settings for scanner V4.
type ScannerV4Persistence struct {
	// Uses a Kubernetes persistent volume claim (PVC) to manage the storage location of persistent data.
	// Recommended for most users.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Persistent volume claim",order=1
	PersistentVolumeClaim *ScannerV4PersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`

	// Stores persistent data on a directory on the host. This is not recommended, and should only
	// be used together with a node selector (only available in YAML view).
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Host path",order=99
	HostPath *HostPathSpec `json:"hostPath,omitempty"`
}

// GetPersistentVolumeClaim returns the configured PVC
func (p *ScannerV4Persistence) GetPersistentVolumeClaim() *ScannerV4PersistentVolumeClaim {
	if p == nil {
		return nil
	}
	return p.PersistentVolumeClaim
}

// GetHostPath returns the configured host path
func (p *ScannerV4Persistence) GetHostPath() string {
	if p == nil {
		return ""
	}
	if p.HostPath == nil {
		return ""
	}

	return pointer.StringDeref(p.HostPath.Path, "")
}

// ScannerV4PersistentVolumeClaim defines PVC-based persistence settings for Scanner V4 DB.
type ScannerV4PersistentVolumeClaim struct {
	// The name of the PVC to manage persistent data. If no PVC with the given name exists, it will be
	// created. Defaults to "scanner-v4-db" if not set.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Claim Name",order=1
	//+kubebuilder:default=scanner-v4-db
	ClaimName *string `json:"claimName,omitempty"`

	// The size of the persistent volume when created through the claim. If a claim was automatically created,
	// this can be used after the initial deployment to resize (grow) the volume (only supported by some
	// storage class controllers).
	//+kubebuilder:validation:Pattern=^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Size",order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Size *string `json:"size,omitempty"`

	// The name of the storage class to use for the PVC. If your cluster is not configured with a default storage
	// class, you must select a value here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage Class",order=3,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass"}
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// ScannerComponentScaling defines replication settings of scanner components.
type ScannerComponentScaling struct {
	// When enabled, the number of component replicas is managed dynamically based on the load, within the limits
	// specified below.
	//+kubebuilder:default=Enabled
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Autoscaling",order=1
	AutoScaling *AutoScalingPolicy `json:"autoScaling,omitempty"`

	// When autoscaling is disabled, the number of replicas will always be configured to match this value.
	//+kubebuilder:default=3
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Default Replicas",order=2
	Replicas *int32 `json:"replicas,omitempty"`

	//+kubebuilder:default=2
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Autoscaling Minimum Replicas",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.autoScaling:Enabled"}
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	//+kubebuilder:default=5
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Autoscaling Maximum Replicas",order=4,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.autoScaling:Enabled"}
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

// AutoScalingPolicy is a type for values of spec.scanner.analyzer.replicas.autoScaling.
// +kubebuilder:validation:Enum=Enabled;Disabled
type AutoScalingPolicy string

const (
	// ScannerAutoScalingEnabled means that scanner autoscaling should be enabled.
	ScannerAutoScalingEnabled AutoScalingPolicy = "Enabled"
	// ScannerAutoScalingDisabled means that scanner autoscaling should be disabled.
	ScannerAutoScalingDisabled AutoScalingPolicy = "Disabled"
)

// Monitoring defines settings for monitoring endpoint.
type Monitoring struct {
	// Expose the monitoring endpoint. A new service, "monitoring",
	// with port 9090, will be created as well as a network policy allowing
	// inbound connections to the port.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	ExposeEndpoint *ExposeEndpoint `json:"exposeEndpoint,omitempty"`
}

// IsEnabled checks whether exposing of endpoint is enabled.
// This method is safe to be used with nil receivers.
func (s *Monitoring) IsEnabled() bool {
	if s == nil || s.ExposeEndpoint == nil {
		return false // disabled by default
	}

	return *s.ExposeEndpoint == ExposeEndpointEnabled
}

// ExposeEndpoint is a type for monitoring sub-struct.
// +kubebuilder:validation:Enum=Enabled;Disabled
type ExposeEndpoint string

const (
	// ExposeEndpointEnabled means that component should expose monitoring port.
	ExposeEndpointEnabled ExposeEndpoint = "Enabled"
	// ExposeEndpointDisabled means that component should not expose monitoring port.
	ExposeEndpointDisabled ExposeEndpoint = "Disabled"
)

// GlobalMonitoring defines settings related to global monitoring. Contrary to
// `Monitoring`, the corresponding Helm flag lives in the global scope `.monitoring`.
type GlobalMonitoring struct {
	OpenShiftMonitoring *OpenShiftMonitoring `json:"openshift,omitempty"`
}

// OpenShiftMonitoring defines settings related to OpenShift Monitoring
type OpenShiftMonitoring struct {
	//+kubebuilder:validation:Default=true
	//+kubebuilder:default=true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled bool `json:"enabled"`
}

// IsOpenShiftMonitoringDisabled returns true if OpenShiftMonitoring is disabled.
// This function is nil safe.
func (m *GlobalMonitoring) IsOpenShiftMonitoringDisabled() bool {
	return m != nil && m.OpenShiftMonitoring != nil && !m.OpenShiftMonitoring.Enabled
}
