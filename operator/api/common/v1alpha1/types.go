//+kubebuilder:object:generate=true

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make generate" in the common directory to regenerate code,
// and run "make manifests" in the "central" and "securedcluster" directories
// to regenerate manifests, after modifying this file
// TODO(ROX-7110): prevent merging PRs if manifests are not up to date.

// CustomizeSpec defines customizations to apply.
type CustomizeSpec struct {
	// Custom labels to set on all objects apart from Pods.
	Labels map[string]string `json:"labels,omitempty"`
	// Custom annotations to set on all objects apart from Pods.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Custom environment variables to set on pods' containers.
	EnvVars []corev1.EnvVar `json:"envVars,omitempty"`
}

// DeploymentSpec defines settings that affect a deployment.
type DeploymentSpec struct {
	NodeSelector map[string]string            `json:"nodeSelector,omitempty"`
	Resources    *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// StackRoxCondition defines a condition for a StackRox custom resource.
type StackRoxCondition struct {
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Type ConditionType `json:"type"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Status ConditionStatus `json:"status"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Reason ConditionReason `json:"reason,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Message string `json:"message,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=status
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
	AdditionalCAs []AdditionalCA `json:"additionalCAs,omitempty"`
}

// LocalSecretReference is a reference to a secret within the same namespace.
type LocalSecretReference struct {
	// The name of the referenced secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	Name string `json:"name"`
}
