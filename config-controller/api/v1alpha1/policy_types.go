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
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
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
	// Notifiers is a list of IDs or names of the notifiers that should be triggered when a violation from this policy is identified.  IDs should be in the form of a UUID and are found through the Central API.
	Notifiers []string `json:"notifiers,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// PolicySections define the violation criteria for this policy.
	PolicySections     []PolicySection      `json:"policySections"`
	MitreAttackVectors []MitreAttackVectors `json:"mitreAttackVectors,omitempty"`
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
	// Cluster is either the name or the ID of the cluster that this scope applies to
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

type SecurityPolicyConditionType string

const (
	Ready      SecurityPolicyConditionType = "Ready"
	Reconciled SecurityPolicyConditionType = "Reconciled"
	Active     SecurityPolicyConditionType = "Active"
)

// SecurityPolicyCondition defines the observed state of SecurityPolicy
type SecurityPolicyCondition struct {
	Type               SecurityPolicyConditionType `json:"type"`
	Status             bool                        `json:"status"`
	Message            string                      `json:"message"`
	LastTransitionTime metav1.Time                 `json:"lastTransitionTime,omitempty"`
}

type SecurityPolicyConditions []SecurityPolicyCondition

type SecurityPolicyStatus struct {
	Condition SecurityPolicyConditions `json:"conditions"`
	PolicyId  string                   `json:"policyId"`
}

// IsValid runs validation checks against the SecurityPolicy spec
func (p SecurityPolicySpec) IsValid() (bool, error) {
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

type CacheType int

const (
	Notifier CacheType = iota
	Cluster
)

func init() {
	SchemeBuilder.Register(&SecurityPolicy{}, &SecurityPolicyList{})
}

func getID(name string, cache map[string]string) (string, error) {
	if _, err := uuid.FromString(name); err == nil {
		return name, nil
	} else if id, exists := cache[name]; exists {
		return id, nil
	} else {
		return "", errors.New("Name not found in cache and passed string was not a valid UUID")
	}
}

func getNotifierID(name string, caches map[CacheType]map[string]string) (string, error) {
	return getID(name, caches[Notifier])
}

func getClusterID(name string, caches map[CacheType]map[string]string) (string, error) {
	return getID(name, caches[Cluster])
}

// ToProtobuf converts the SecurityPolicy spec into policy proto
func (p SecurityPolicySpec) ToProtobuf(caches map[CacheType]map[string]string) (*storage.Policy, error) {
	proto := storage.Policy{
		Name:          p.PolicyName,
		Description:   p.Description,
		Rationale:     p.Rationale,
		Remediation:   p.Remediation,
		Disabled:      p.Disabled,
		Categories:    p.Categories,
		PolicyVersion: policyversion.CurrentVersion().String(),
		Source:        storage.PolicySource_DECLARATIVE,
	}

	proto.Notifiers = make([]string, 0, len(p.Notifiers))
	for _, notifier := range p.Notifiers {
		id, err := getNotifierID(notifier, caches)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Notifier '%s' does not exist", notifier))
		}
		proto.Notifiers = append(proto.Notifiers, id)
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
				return nil, errors.Wrapf(err, "Error parsing timestamp '%s'", exclusion.Expiration)
			}
			protoExclusion.Expiration = protoTS
		}

		if exclusion.Deployment != (Deployment{}) {
			protoExclusion.Deployment = &storage.Exclusion_Deployment{
				Name: exclusion.Deployment.Name,
			}

			scope := exclusion.Deployment.Scope
			if scope != (Scope{}) {
				clusterID, err := getClusterID(scope.Cluster, caches)
				if err != nil {
					return nil, errors.New(fmt.Sprintf("Cluster '%s' does not exist", scope.Cluster))
				}
				protoExclusion.Deployment.Scope = &storage.Scope{
					Cluster:   clusterID,
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
		clusterID, err := getClusterID(scope.Cluster, caches)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Cluster '%s' does not exist", scope.Cluster))
		}
		protoScope := &storage.Scope{
			Cluster:   clusterID,
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

	return &proto, nil
}

func (s *SecurityPolicyConditions) UpdateCondition(sType SecurityPolicyConditionType, newCondition SecurityPolicyCondition) {
	for i, st := range *s {
		if st.Type != sType {
			continue
		}
		if st.Status != newCondition.Status {
			newCondition.LastTransitionTime = metav1.Time{}
		} else {
			newCondition.LastTransitionTime = st.LastTransitionTime
		}
		(*s)[i] = newCondition
		return
	}
}

func (s *SecurityPolicyConditions) GetCondition(sType SecurityPolicyConditionType) *SecurityPolicyCondition {
	for _, st := range *s {
		if st.Type == sType {
			return &st
		}
	}
	return nil
}
