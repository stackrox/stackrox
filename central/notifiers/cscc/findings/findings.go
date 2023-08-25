// Package findings defines types for APIs related to Finding messages.
package findings

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// State indicates whether a finding is active.
type State string

// These State values are defined in the Google Cloud SCC API.
//
// Note: For ATTEMPTED alerts, the INACTIVE state seems to be a better fit to convey that StackRox stopped (resolved)
// an impending threat/violation by blocking the action.
const (
	StateActive   State = "ACTIVE"
	StateInactive State = "INACTIVE"
)

// Severity indicates severity of alert
type Severity string

// Severity values are defined in google cloud scc api
const (
	High     Severity = "HIGH"
	Low      Severity = "LOW"
	Medium   Severity = "MEDIUM"
	Critical Severity = "CRITICAL"
	Default  Severity = "SEVERITY_UNSPECIFIED"
)

// A Finding represents a single Finding created by StackRox (as a Source).
type Finding struct {
	ID           string                 `json:"name,omitempty"`
	Parent       string                 `json:"parent,omitempty"`
	ResourceName string                 `json:"resourceName,omitempty"`
	State        State                  `json:"state,omitempty"`
	Category     string                 `json:"category,omitempty"`
	URL          string                 `json:"externalUri,omitempty"`
	Properties   map[string]interface{} `json:"sourceProperties,omitempty"`
	Timestamp    string                 `json:"eventTime"`
	Severity     Severity               `json:"severity,omitempty"`
	// The time at which the event took place. For example, if the finding represents an open firewall it would capture the time the open firewall was detected.
	// A timestamp in RFC3339 UTC "Zulu" format, accurate to nanoseconds. Example: "2014-10-02T15:01:23.045123456Z".
	// See https://cloud.google.com/security-command-center/docs/reference/rest/v1beta1/organizations.sources.findings#Finding
}

// A ClusterID creates a structured ID for the ResourceName field.
type ClusterID struct {
	Project string
	Zone    string
	Name    string
}

// ResourceName is the format needed for the ResourceName field.
func (c ClusterID) ResourceName() string {
	return fmt.Sprintf("//container.googleapis.com/projects/%s/zones/%s/clusters/%s", c.Project, c.Zone, c.Name)
}

// An Enforcement object reports that an enforcement action has been taken.
type Enforcement struct {
	Action    string               `json:"action,omitempty"`
	Message   string               `json:"message,omitempty"`
	Timestamp *timestamp.Timestamp `json:"timestamp,omitempty"`
}

// Properties includes various values, by key, for a new Finding.
type Properties struct {

	// These fields are custom and defined by StackRox.
	Namespace      string `json:"namespace,omitempty"`
	Service        string `json:"service,omitempty"`
	DeploymentType string `json:"deployment_type,omitempty"`
	ResourceType   string `json:"resource_type,omitempty"`

	EnforcementActions []Enforcement `json:"enforcement_actions,omitempty"`
	Summary            string        `json:"summary,omitempty"`
}

// Map changes the Properties struct into an untyped map for API usage.
func (p Properties) Map() map[string]interface{} {
	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}
	return m
}
