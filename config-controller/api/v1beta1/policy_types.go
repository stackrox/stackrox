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

package v1beta1

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/protocompat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/stackrox/rox/config-controller/api/v1alpha1"
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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[^\n\r\$]{5,128}$`
	PolicyName string `json:"policyName,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[^\$]{0,800}$`
	Description string `json:"description,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	// +kubebuilder:validation:MinItems=1
	Categories []string `json:"categories,omitempty"`
	// +kubebuilder:validation:MinItems=1
	LifecycleStages []LifecycleStage `json:"lifecycleStages,omitempty"`
	EventSource     EventSource      `json:"eventSource,omitempty"`
	Exclusions      []Exclusion      `json:"exclusions,omitempty"`
	Scope           []Scope          `json:"scope,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=UNSET_SEVERITY;LOW_SEVERITY;MEDIUM_SEVERITY;HIGH_SEVERITY;CRITICAL_SEVERITY
	Severity           string              `json:"severity,omitempty"`
	EnforcementActions []EnforcementAction `json:"enforcementActions,omitempty"`
	Notifiers          []string            `json:"notifiers,omitempty"`
	// +kubebuilder:validation:MinItems=1
	Criteria           []Criterion          `json:"criteria,omitempty"`
	MitreAttackVectors []MitreAttackVectors `json:"mitreAttackVectors,omitempty"`
	CriteriaLocked     bool                 `json:"criteriaLocked,omitempty"`
	MitreVectorsLocked bool                 `json:"mitreVectorsLocked,omitempty"`
	IsDefault          bool                 `json:"isDefault,omitempty"`
	TestField          string               `json:"testField,omitempty"`
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

type Criterion struct {
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
	PolicyId string `json:"policyId"`
}

// IsValid runs validation checks against the SecurityPolicy spec
func (p SecurityPolicySpec) IsValid() (bool, error) {
	if p.IsDefault {
		return false, errors.New("isDefault must be false")
	}
	return true, nil
}

// ToProtobuf converts the SecurityPolicy spec into policy proto
func (p SecurityPolicySpec) ToProtobuf() *storage.Policy {
	proto := storage.Policy{
		Name:               p.PolicyName,
		Description:        p.Description,
		Rationale:          p.Rationale,
		Remediation:        p.Remediation,
		Disabled:           p.Disabled,
		Categories:         p.Categories,
		Notifiers:          p.Notifiers,
		PolicyVersion:      policyversion.CurrentVersion().String(),
		CriteriaLocked:     p.CriteriaLocked,
		MitreVectorsLocked: p.MitreVectorsLocked,
		IsDefault:          p.IsDefault,
		Source:             storage.PolicySource_DECLARATIVE,
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

		if exclusion.Expiration != "" {
			protoTS, err := protocompat.ParseRFC3339NanoTimestamp(exclusion.Expiration)
			if err != nil {
				return nil
			}
			protoExclusion.Expiration = protoTS

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

	protoSection := &storage.PolicySection{SectionName: "Section 1"}

	for _, criterion := range p.Criteria {
		protoGroup := &storage.PolicyGroup{
			FieldName: criterion.FieldName,
			Negate:    criterion.Negate,
		}

		val, found = storage.BooleanOperator_value[criterion.BooleanOperator]
		if found {
			protoGroup.BooleanOperator = storage.BooleanOperator(val)
		}

		for _, value := range criterion.Values {
			protoValue := &storage.PolicyValue{
				Value: value.Value,
			}
			protoGroup.Values = append(protoGroup.Values, protoValue)
		}
		protoSection.PolicyGroups = append(protoSection.PolicyGroups, protoGroup)
	}
	proto.PolicySections = []*storage.PolicySection{protoSection}

	for _, mitreAttackVectors := range p.MitreAttackVectors {
		protoMitreAttackVetor := &storage.Policy_MitreAttackVectors{
			Tactic:     mitreAttackVectors.Tactic,
			Techniques: mitreAttackVectors.Techniques,
		}

		proto.MitreAttackVectors = append(proto.MitreAttackVectors, protoMitreAttackVetor)
	}

	return &proto
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

func (src *SecurityPolicy) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.SecurityPolicy)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.PolicyName = src.Spec.PolicyName
	dst.Spec.Description = src.Spec.Description
	dst.Spec.Rationale = src.Spec.Rationale
	dst.Spec.Remediation = src.Spec.Remediation
	dst.Spec.Disabled = src.Spec.Disabled
	dst.Spec.Categories = src.Spec.Categories
	dst.Spec.Notifiers = src.Spec.Notifiers
	dst.Spec.CriteriaLocked = src.Spec.CriteriaLocked
	dst.Spec.MitreVectorsLocked = src.Spec.MitreVectorsLocked
	dst.Spec.IsDefault = src.Spec.IsDefault

	for _, ls := range src.Spec.LifecycleStages {
		dst.Spec.LifecycleStages = append(dst.Spec.LifecycleStages, v1alpha1.LifecycleStage(ls))
	}

	for _, exclusion := range src.Spec.Exclusions {
		dstExclusion := v1alpha1.Exclusion{
			Name: exclusion.Name,
		}

		dstExclusion.Expiration = exclusion.Expiration

		if exclusion.Deployment != (Deployment{}) {
			dstExclusion.Deployment = v1alpha1.Deployment{
				Name: exclusion.Deployment.Name,
			}

			scope := exclusion.Deployment.Scope
			if scope != (Scope{}) {
				dstExclusion.Deployment.Scope = v1alpha1.Scope{
					Cluster:   scope.Cluster,
					Namespace: scope.Namespace,
				}
			}

			if scope.Label != (Label{}) {
				dstExclusion.Deployment.Scope.Label = v1alpha1.Label{
					Key:   scope.Label.Key,
					Value: scope.Label.Value,
				}
			}

		}

		dst.Spec.Exclusions = append(dst.Spec.Exclusions, dstExclusion)
	}

	for _, scope := range src.Spec.Scope {
		dstScope := v1alpha1.Scope{
			Cluster:   scope.Cluster,
			Namespace: scope.Namespace,
		}

		if scope.Label != (Label{}) {
			dstScope.Label = v1alpha1.Label{
				Key:   scope.Label.Key,
				Value: scope.Label.Value,
			}
		}

		dst.Spec.Scope = append(dst.Spec.Scope, dstScope)
	}

	dst.Spec.Severity = src.Spec.Severity

	dst.Spec.EventSource = v1alpha1.EventSource(src.Spec.EventSource)

	for _, ea := range src.Spec.EnforcementActions {
		dst.Spec.EnforcementActions = append(dst.Spec.EnforcementActions, v1alpha1.EnforcementAction(ea))
	}

	dstSection := v1alpha1.PolicySection{SectionName: "Section 1"}

	for _, criterion := range src.Spec.Criteria {
		dstGroup := v1alpha1.PolicyGroup{
			FieldName: criterion.FieldName,
			Negate:    criterion.Negate,
		}

		dstGroup.BooleanOperator = criterion.BooleanOperator

		for _, value := range criterion.Values {
			dstValue := v1alpha1.PolicyValue{
				Value: value.Value,
			}
			dstGroup.Values = append(dstGroup.Values, dstValue)
		}
		dstSection.PolicyGroups = append(dstSection.PolicyGroups, dstGroup)
	}
	dst.Spec.PolicySections = []v1alpha1.PolicySection{dstSection}

	for _, mitreAttackVectors := range src.Spec.MitreAttackVectors {
		dstMitreAttackVetor := v1alpha1.MitreAttackVectors{
			Tactic:     mitreAttackVectors.Tactic,
			Techniques: mitreAttackVectors.Techniques,
		}

		dst.Spec.MitreAttackVectors = append(dst.Spec.MitreAttackVectors, dstMitreAttackVetor)
	}
	return nil
}

func (dst *SecurityPolicy) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.SecurityPolicy)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.PolicyName = src.Spec.PolicyName
	dst.Spec.Description = src.Spec.Description
	dst.Spec.Rationale = src.Spec.Rationale
	dst.Spec.Remediation = src.Spec.Remediation
	dst.Spec.Disabled = src.Spec.Disabled
	dst.Spec.Categories = src.Spec.Categories
	dst.Spec.Notifiers = src.Spec.Notifiers
	dst.Spec.CriteriaLocked = src.Spec.CriteriaLocked
	dst.Spec.MitreVectorsLocked = src.Spec.MitreVectorsLocked
	dst.Spec.IsDefault = src.Spec.IsDefault

	for _, ls := range src.Spec.LifecycleStages {
		dst.Spec.LifecycleStages = append(dst.Spec.LifecycleStages, LifecycleStage(ls))
	}

	for _, exclusion := range src.Spec.Exclusions {
		dstExclusion := Exclusion{
			Name: exclusion.Name,
		}

		dstExclusion.Expiration = exclusion.Expiration

		if exclusion.Deployment != (v1alpha1.Deployment{}) {
			dstExclusion.Deployment = Deployment{
				Name: exclusion.Deployment.Name,
			}

			scope := exclusion.Deployment.Scope
			if scope != (v1alpha1.Scope{}) {
				dstExclusion.Deployment.Scope = Scope{
					Cluster:   scope.Cluster,
					Namespace: scope.Namespace,
				}
			}

			if scope.Label != (v1alpha1.Label{}) {
				dstExclusion.Deployment.Scope.Label = Label{
					Key:   scope.Label.Key,
					Value: scope.Label.Value,
				}
			}

		}

		dst.Spec.Exclusions = append(dst.Spec.Exclusions, dstExclusion)
	}

	for _, scope := range src.Spec.Scope {
		dstScope := Scope{
			Cluster:   scope.Cluster,
			Namespace: scope.Namespace,
		}

		if scope.Label != (v1alpha1.Label{}) {
			dstScope.Label = Label{
				Key:   scope.Label.Key,
				Value: scope.Label.Value,
			}
		}

		dst.Spec.Scope = append(dst.Spec.Scope, dstScope)
	}

	dst.Spec.Severity = src.Spec.Severity

	dst.Spec.EventSource = EventSource(src.Spec.EventSource)

	for _, ea := range src.Spec.EnforcementActions {
		dst.Spec.EnforcementActions = append(dst.Spec.EnforcementActions, EnforcementAction(ea))
	}

	for _, section := range src.Spec.PolicySections {
		for _, group := range section.PolicyGroups {
			criterion := Criterion{
				FieldName: group.FieldName,
				Negate:    group.Negate,
			}

			criterion.BooleanOperator = group.BooleanOperator

			for _, value := range group.Values {
				dstValue := PolicyValue{
					Value: value.Value,
				}
				criterion.Values = append(criterion.Values, dstValue)
			}
			dst.Spec.Criteria = append(dst.Spec.Criteria, criterion)
		}
	}

	for _, mitreAttackVectors := range src.Spec.MitreAttackVectors {
		dstMitreAttackVetor := MitreAttackVectors{
			Tactic:     mitreAttackVectors.Tactic,
			Techniques: mitreAttackVectors.Techniques,
		}

		dst.Spec.MitreAttackVectors = append(dst.Spec.MitreAttackVectors, dstMitreAttackVetor)
	}
	return nil
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
