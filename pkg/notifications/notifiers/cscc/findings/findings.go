// Package findings defines types for APIs related to SourceFinding messages.
package findings

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	// SourceID is a value defined by Google and must be provided in the sourceId field.
	SourceID = "STACKROX"
)

// A CreateFindingMessage is provided to the CreateFindings API.
type CreateFindingMessage struct {
	Finding SourceFinding `json:"sourceFinding,omitempty"`
}

// A SourceFinding represents a single Finding created by StackRox (as a Source).
type SourceFinding struct {
	ID         string                 `json:"id,omitempty"`
	Category   string                 `json:"category,omitempty"`
	AssetIDs   []string               `json:"assetIds,omitempty"`  //`json:"asset_ids,omitempty"`
	SourceID   string                 `json:"sourceId,omitempty"`  //`json:"source_id,omitempty"`
	Timestamp  *timestamp.Timestamp   `json:"eventTime,omitempty"` //`json:"event_time,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	// Other fields are ignored in requests
}

// A ClusterID creates a structured ID for the AssetIDs field.
type ClusterID struct {
	Org     string
	Project string
	ID      string
}

// AssetID is the format needed for the AssetIDs field.
func (c ClusterID) AssetID() string {
	return fmt.Sprintf("organizations/%s/%s/cluster/%s", c.Org, c.Project, c.ID)
}

// An Enforcement object reports that an enforcement action has been taken.
type Enforcement struct {
	Action    string               `json:"action,omitempty"`
	Message   string               `json:"message,omitempty"`
	Timestamp *timestamp.Timestamp `json:"timestamp,omitempty"`
}

// Properties includes various values, by key, for a new Finding.
type Properties struct {
	// These fields are required by Google, at least for v1alpha3:
	SCCCategory     string `json:"scc_category,omitempty"`
	SCCStatus       string `json:"scc_status,omitempty"`        // Must be "active" or "inactive"
	SCCSeverity     string `json:"scc_severity,omitempty"`      // Must be critical, high, medium, low, or info.
	SCCSourceStatus string `json:"scc_source_status,omitempty"` // a partner generated string e.g. false-positive, ignore, mute

	// These fields are custom and defined by StackRox.
	Namespace      string `json:"namespace,omitempty"`
	Service        string `json:"service,omitempty"`
	DeploymentType string `json:"deployment_type,omitempty"`
	Container      string `json:"container,omitempty"`
	ContainerID    string `json:"container_id,omitempty"`

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
