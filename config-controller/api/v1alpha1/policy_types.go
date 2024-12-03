/*
Copyright 2024.

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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=DEPLOY;BUILD;RUNTIME
type LifecycleStage string

// +kubebuilder:validation:Enum=NOT_APPLICABLE;DEPLOYMENT_EVENT;AUDIT_LOG_EVENT
type EventSource string

// +kubebuilder:validation:Enum=UNSET_ENFORCEMENT;SCALE_TO_ZERO_ENFORCEMENT;UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT;KILL_POD_ENFORCEMENT;FAIL_BUILD_ENFORCEMENT;FAIL_KUBE_REQUEST_ENFORCEMENT;FAIL_DEPLOYMENT_CREATE_ENFORCEMENT;FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT
type EnforcementAction string

// SecurityPolicySpec defines the desired state of SecurityPolicy
type SecurityPolicySpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[^\n\r\$]{5,128}$`
	// PolicyName is the name of the policy as it appears in the API and UI.  Note that changing this value will rename the policy as stored in the database.  This field must be unique.
	PolicyName string `json:"policyName"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[^\$]{0,800}$`
	// Description is a free-form text description of this policy.
	Description string `json:"description,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
	// Remediation describes how to remediate a violation of this policy.
	Remediation string `json:"remediation,omitempty"`
	// Disabled toggles whether or not this policy will be executing and actively firing alerts.
	Disabled bool `json:"disabled,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// Categories is a list of categories that this policy falls under.  Category names must already exist in Central.
	Categories []string `json:"categories"`
	// +kubebuilder:validation:MinItems=1
	// LifecycleStages describes which policy lifecylce stages this policy applies to.  Choices are DEPLOY, BUILD, and RUNTIME.
	LifecycleStages []LifecycleStage `json:"lifecycleStages"`
	// EventSource describes which events should trigger execution of this policy
	EventSource EventSource `json:"eventSource,omitempty"`
	// Exclusions define deployments or images that should be excluded from this policy.
	Exclusions []Exclusion `json:"exclusions,omitempty"`
	// Scope defines clusters, namespaces, and deployments that should be included in this policy.  No scopes defined includes everything.
	Scope []Scope `json:"scope,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=UNSET_SEVERITY;LOW_SEVERITY;MEDIUM_SEVERITY;HIGH_SEVERITY;CRITICAL_SEVERITY
	// Severity defines how severe a violation from this policy is.  Possible values are UNSET_SEVERITY, LOW_SEVERITY, MEDIUM_SEVERITY, HIGH_SEVERITY, and CRITICAL_SEVERITY.
	Severity string `json:"severity"`
	// Enforcement lists the enforcement actions to take when a violation from this policy is identified.  Possible value are UNSET_ENFORCEMENT, SCALE_TO_ZERO_ENFORCEMENT, UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, KILL_POD_ENFORCEMENT, FAIL_BUILD_ENFORCEMENT, FAIL_KUBE_REQUEST_ENFORCEMENT, FAIL_DEPLOYMENT_CREATE_ENFORCEMENT, and. FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT.
	EnforcementActions []EnforcementAction `json:"enforcementActions,omitempty"`
	// Notifiers is a list of IDs of the notifiers that should be triggered when a violation from this policy is identified.  IDs should be in the form of a UUID and are found through the Central API.
	Notifiers []string `json:"notifiers,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// PolicySections define the violation criteria for this policy.
	PolicySections     []PolicySection      `json:"policySections"`
	MitreAttackVectors []MitreAttackVectors `json:"mitreAttackVectors,omitempty"`
	// Read-only field. If true, the policy's criteria fields are rendered read-only.
	CriteriaLocked bool `json:"criteriaLocked,omitempty"`
	// Read-only field. If true, the policy's MITRE ATT&CK fields are rendered read-only.
	MitreVectorsLocked bool `json:"mitreVectorsLocked,omitempty"`
	// Read-only field. Indicates the policy is a default policy if true and a custom policy if false.
	IsDefault bool `json:"isDefault,omitempty"`
}

type Exclusion struct {
	Name       string     `json:"name,omitempty"`
	Deployment Deployment `json:"deployment,omitempty"`
	Image      Image      `json:"image,omitempty"`
	// +optional
	// +kubebuilder:validation:Format="date-time"
	Expiration string `json:"expiration,omitempty"`
}

type Deployment struct {
	Name  string `json:"name,omitempty"`
	Scope Scope  `json:"scope,omitempty"`
}

type Image struct {
	Name string `json:"name,omitempty"`
}

type Scope struct {
	Cluster   string `json:"cluster,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Label     Label  `json:"label,omitempty"`
}

type Label struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type PolicySection struct {
	// SectionName is a user-friendly name for this section of policies
	SectionName string `json:"sectionName,omitempty"`
	// PolicyGroups is the set of policies groups that make up this section.  Each group can be considered an individual criterion.
	PolicyGroups []PolicyGroup `json:"policyGroups"`
}

type PolicyGroup struct {

	// FieldName defines which field on a deployment or image this PolicyGroup evaluates.  See https://docs.openshift.com/acs/operating/manage-security-policies.html#policy-criteria_manage-security-policies for a complete list of possible values.
	FieldName string `json:"fieldName"`
	// +kubebuilder:validation:Enum=OR;AND
	// BooleanOperator determines if the values are combined with an OR or an AND.  Defaults to OR.
	BooleanOperator string `json:"booleanOperator,omitempty"`
	// Negate determines if the evaluation of this PolicyGroup is negated.  Default to false.
	Negate bool `json:"negate,omitempty"`
	// Values is the list of values for the specified field
	Values []PolicyValue `json:"values,omitempty"`
}

type PolicyValue struct {
	// Value is simply the string value
	Value string `json:"value,omitempty"`
}

type MitreAttackVectors struct {
	Tactic     string   `json:"tactic,omitempty"`
	Techniques []string `json:"techniques,omitempty"`
}

// SecurityPolicyStatus defines the observed state of SecurityPolicy
type SecurityPolicyStatus struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
	PolicyId string `json:"policyId"`
}

// IsValid runs validation checks against the SecurityPolicy spec
func (p SecurityPolicySpec) IsValid() (bool, error) {
	if p.IsDefault {
		return false, errors.New("isDefault must be false")
	}
	return true, nil
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=sp
// +kubebuilder:subresource:status

// SecurityPolicy is the Schema for the policies API
type SecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityPolicySpec   `json:"spec,omitempty"`
	Status SecurityPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityPolicyList contains a list of Policy
type SecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityPolicy{}, &SecurityPolicyList{})
}
