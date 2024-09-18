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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stackrox/rox/generated/storage"
)

// +kubebuilder:validation:Enum=DEPLOY;BUILD;RUNTIME
type LifecycleStage string

// +kubebuilder:validation:Enum=NOT_APPLICABLE;DEPLOYMENT_EVENT;AUDIT_LOG_EVENT
type EventSource string

// +kubebuilder:validation:Enum=UNSET_ENFORCEMENT;SCALE_TO_ZERO_ENFORCEMENT;UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT;KILL_POD_ENFORCEMENT;FAIL_BUILD_ENFORCEMENT;FAIL_KUBE_REQUEST_ENFORCEMENT;FAIL_DEPLOYMENT_CREATE_ENFORCEMENT;FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT
type EnforcementAction string

// SecurityPolicySpec defines the desired state of SecurityPolicy
type SecurityPolicySpec struct {
	Description string   `json:"description,omitempty"`
	Rationale   string   `json:"rationale,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Disabled    bool     `json:"disabled,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	// +kubebuilder:validation:MinItems=1
	LifecycleStages []LifecycleStage `json:"lifecycleStages,omitempty"`
	EventSource     EventSource      `json:"eventSource,omitempty"`
	Exclusions      []Exclusion      `json:"exclusions,omitempty"`
	Scope           []Scope          `json:"scope,omitempty"`
	// +kubebuilder:validation:Enum=UNSET_SEVERITY;LOW_SEVERITY;MEDIUM_SEVERITY;HIGH_SEVERITY;CRITICAL_SEVERITY
	Severity           string               `json:"severity,omitempty"`
	EnforcementActions []EnforcementAction  `json:"enforcementActions,omitempty"`
	Notifiers          []string             `json:"notifiers,omitempty"`
	PolicyVersion      string               `json:"policyVersion,omitempty"`
	PolicySections     []PolicySection      `json:"policySections,omitempty"`
	MitreAttackVectors []MitreAttackVectors `json:"mitreAttackVectors,omitempty"`
	CriteriaLocked     bool                 `json:"criteriaLocked,omitempty"`
	MitreVectorsLocked bool                 `json:"mitreVectorsLocked,omitempty"`
	IsDefault          bool                 `json:"isDefault,omitempty"`
}

func (p SecurityPolicySpec) ToProtobuf() *storage.Policy {
	proto := storage.Policy{
		Description:        p.Description,
		Rationale:          p.Rationale,
		Remediation:        p.Remediation,
		Disabled:           p.Disabled,
		Categories:         p.Categories,
		Notifiers:          p.Notifiers,
		PolicyVersion:      p.PolicyVersion,
		CriteriaLocked:     p.CriteriaLocked,
		MitreVectorsLocked: p.MitreVectorsLocked,
		IsDefault:          p.IsDefault,
	}

	for _, ls := range p.LifecycleStages {
		val, found := storage.LifecycleStage_value[string(ls)]
		if found {
			proto.LifecycleStages = append(proto.LifecycleStages, storage.LifecycleStage(val))
		}
	}

	for _, exclusion := range p.Exclusions {
		protoExclusion := storage.Exclusion{
			Name: exclusion.Name,
		}

		if exclusion.Deployment != (Deployment{}) {
			protoExclusion.Deployment = &storage.Exclusion_Deployment{
				Name: exclusion.Deployment.Name,
			}

			scope := exclusion.Deployment.Scope
			if scope != (Scope{}) {
				protoExclusion.Deployment.Scope = &storage.Scope{
					Cluster:   scope.Cluster,
					Namespace: scope.Namespace,
				}
			}

			if scope.Label != (Label{}) {
				protoExclusion.Deployment.Scope.Label = &storage.Scope_Label{
					Key:   scope.Label.Key,
					Value: scope.Label.Value,
				}
			}
		}

		proto.Exclusions = append(proto.Exclusions, &protoExclusion)
	}

	for _, scope := range p.Scope {
		protoScope := &storage.Scope{
			Cluster:   scope.Cluster,
			Namespace: scope.Namespace,
		}

		if scope.Label != (Label{}) {
			protoScope.Label = &storage.Scope_Label{
				Key:   scope.Label.Key,
				Value: scope.Label.Value,
			}
		}

		proto.Scope = append(proto.Scope, protoScope)
	}

	val, found := storage.Severity_value[p.Severity]
	if found {
		proto.Severity = storage.Severity(val)
	}

	val, found = storage.EventSource_value[string(p.EventSource)]
	if found {
		proto.EventSource = storage.EventSource(val)
	}

	for _, ea := range p.EnforcementActions {
		val, found := storage.EnforcementAction_value[string(ea)]
		if found {
			proto.EnforcementActions = append(proto.EnforcementActions, storage.EnforcementAction(val))
		}
	}

	for _, section := range p.PolicySections {
		protoSection := &storage.PolicySection{
			SectionName: section.SectionName,
		}

		for _, group := range section.PolicyGroups {
			protoGroup := &storage.PolicyGroup{
				FieldName: group.FieldName,
				Negate:    group.Negate,
			}

			val, found = storage.BooleanOperator_value[group.BooleanOperator]
			if found {
				protoGroup.BooleanOperator = storage.BooleanOperator(val)
			}

			for _, value := range group.Values {
				protoValue := &storage.PolicyValue{
					Value: value.Value,
				}
				protoGroup.Values = append(protoGroup.Values, protoValue)
			}
			protoSection.PolicyGroups = append(protoSection.PolicyGroups, protoGroup)
		}
		proto.PolicySections = append(proto.PolicySections, protoSection)
	}

	for _, mitreAttackVectors := range p.MitreAttackVectors {
		protoMitreAttackVetor := &storage.Policy_MitreAttackVectors{
			Tactic:     mitreAttackVectors.Tactic,
			Techniques: mitreAttackVectors.Techniques,
		}

		proto.MitreAttackVectors = append(proto.MitreAttackVectors, protoMitreAttackVetor)
	}

	return &proto
}

type Exclusion struct {
	Name       string     `json:"name,omitempty"`
	Deployment Deployment `json:"deployment,omitempty"`
	Image      Image      `json:"image,omitempty"`
	//TODO: Expiration
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
	SectionName  string        `json:"sectionName,omitempty"`
	PolicyGroups []PolicyGroup `json:"policyGroups,omitempty"`
}

type PolicyGroup struct {
	FieldName string `json:"fieldName,omitempty"`
	// +kubebuilder:validation:Enum=OR;AND
	BooleanOperator string        `json:"booleanOperator,omitempty"`
	Negate          bool          `json:"negate,omitempty"`
	Values          []PolicyValue `json:"values,omitempty"`
}

type PolicyValue struct {
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
}

// +kubebuilder:object:root=true
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
