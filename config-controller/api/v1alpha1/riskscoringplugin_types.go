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
)

// RiskScoringPluginSpec defines the desired state of RiskScoringPlugin
type RiskScoringPluginSpec struct {
	// Type specifies the plugin execution model.
	// Currently only "builtin" is supported.
	// +kubebuilder:validation:Enum=builtin
	// +kubebuilder:default:=builtin
	Type string `json:"type"`

	// Enabled controls whether this plugin contributes to risk scoring.
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled,omitempty"`

	// Weight is a multiplier applied to the plugin's raw score.
	// Specified as a string to avoid float precision issues (e.g., "1.0", "2.5").
	// +kubebuilder:default:="1.0"
	Weight string `json:"weight,omitempty"`

	// Priority determines execution order. Lower values run earlier.
	// +kubebuilder:default:=1000
	Priority int32 `json:"priority,omitempty"`

	// Builtin specifies configuration for a built-in plugin.
	// Required when Type is "builtin".
	Builtin *BuiltinPluginSpec `json:"builtin,omitempty"`
}

// BuiltinPluginSpec specifies configuration for a built-in risk scoring plugin.
type BuiltinPluginSpec struct {
	// Name references a compiled plugin.
	// +kubebuilder:validation:Enum=policy-violations;process-baselines;vulnerabilities;service-config;port-exposure;risky-components;component-count;image-age
	Name string `json:"name"`

	// Parameters are plugin-specific configuration values.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// RiskScoringPluginConditionType represents a condition type for RiskScoringPlugin.
type RiskScoringPluginConditionType string

const (
	// RiskScoringPluginSynced indicates the plugin config was synced to Central.
	RiskScoringPluginSynced RiskScoringPluginConditionType = "Synced"
)

// RiskScoringPluginStatus defines the observed state of RiskScoringPlugin
type RiskScoringPluginStatus struct {
	// Conditions represent the latest available observations of the plugin's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ConfigID is the ID assigned by Central for this plugin configuration.
	// +optional
	ConfigID string `json:"configId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=rsp
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Plugin",type=string,JSONPath=`.spec.builtin.name`
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`
// +kubebuilder:printcolumn:name="Weight",type=number,JSONPath=`.spec.weight`
// +kubebuilder:printcolumn:name="Priority",type=integer,JSONPath=`.spec.priority`

// RiskScoringPlugin is the Schema for the riskscoringplugins API.
// It allows configuration of risk scoring plugins in StackRox Central.
type RiskScoringPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RiskScoringPluginSpec   `json:"spec,omitempty"`
	Status RiskScoringPluginStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RiskScoringPluginList contains a list of RiskScoringPlugin
type RiskScoringPluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RiskScoringPlugin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RiskScoringPlugin{}, &RiskScoringPluginList{})
}
