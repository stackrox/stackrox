package cscc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// A clusterID creates a structured ID for the ResourceName field.
type clusterID struct {
	Project string
	Zone    string
	Name    string
}

// ResourceName is the format needed for the ResourceName field.
func (c clusterID) ResourceName() string {
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

func convertAlertToFinding(alert *storage.Alert, sourceID string, notifierEndpoint string,
	providerMetadata *storage.ProviderMetadata) (string, *securitycenterpb.Finding, error) {
	findingID := convertAlertUUID(alert.GetId())

	finding := &securitycenterpb.Finding{
		Name:   fmt.Sprintf("%s/findings/%s", sourceID, findingID),
		Parent: sourceID,
		ResourceName: clusterID{
			Project: providerMetadata.GetGoogle().GetProject(),
			Zone:    providerMetadata.GetZone(),
			Name:    providerMetadata.GetGoogle().GetClusterName(),
		}.ResourceName(),
		Category:    alert.GetPolicy().GetName(),
		ExternalUri: notifiers.AlertLink(notifierEndpoint, alert),
		EventTime:   timestamppb.New(protoconv.ConvertTimestampToTimeOrNow(alert.GetTime())),
		Severity:    convertSeverity(alert.GetPolicy().GetSeverity()),
		State: utils.IfThenElse(alert.GetState() == storage.ViolationState_ATTEMPTED,
			securitycenterpb.Finding_INACTIVE,
			securitycenterpb.Finding_ACTIVE),
	}

	var props *Properties
	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		props = &Properties{

			Namespace:      alert.GetDeployment().GetNamespace(),
			Service:        alert.GetDeployment().GetName(),
			DeploymentType: alert.GetDeployment().GetType(),

			EnforcementActions: convertEnforcement(alert),
			Summary:            convertAlertDescription(alert),
		}
	case *storage.Alert_Resource_:
		props = &Properties{

			Namespace:    alert.GetResource().GetNamespace(),
			Service:      alert.GetResource().GetName(),
			ResourceType: alert.GetResource().GetResourceType().String(),

			EnforcementActions: convertEnforcement(alert),
			Summary:            convertAlertDescription(alert),
		}
	}

	if props != nil {
		protoStruct, err := structpb.NewStruct(props.Map())
		if err != nil {
			return "", nil, errors.Wrap(err, "creating source properties")
		}
		finding.SourceProperties = protoStruct.GetFields()
	}

	return findingID, finding, nil
}

func convertSeverity(s storage.Severity) securitycenterpb.Finding_Severity {
	switch s {
	case storage.Severity_LOW_SEVERITY:
		return securitycenterpb.Finding_LOW
	case storage.Severity_MEDIUM_SEVERITY:
		return securitycenterpb.Finding_MEDIUM
	case storage.Severity_HIGH_SEVERITY:
		return securitycenterpb.Finding_HIGH
	case storage.Severity_CRITICAL_SEVERITY:
		return securitycenterpb.Finding_CRITICAL
	default:
		return securitycenterpb.Finding_SEVERITY_UNSPECIFIED
	}
}

func convertEnforcementAction(a storage.EnforcementAction) string {
	switch a {
	case storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:
		return "Scaled to zero replicas"
	case storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT:
		return "Unsatisfiable node constraint added to prevent deployment"
	case storage.EnforcementAction_FAIL_DEPLOYMENT_CREATE_ENFORCEMENT:
		return "Blocked deployment create"
	case storage.EnforcementAction_FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT:
		return "Blocked deployment update"
	case storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT:
		return "Blocked kubernetes operation"
	default:
		return a.String()
	}
}

// Cloud SCC requires that Finding IDs be alphanumeric (no special characters)
// and 1-32 characters long. UUIDs are 32 characters if you remove hyphens.
func convertAlertUUID(u string) string {
	return strings.Replace(u, "-", "", -1)
}

func convertEnforcement(alert *storage.Alert) []Enforcement {
	if alert.GetEnforcement().GetAction() == storage.EnforcementAction_UNSET_ENFORCEMENT {
		return nil
	}
	return []Enforcement{
		{
			Action:  convertEnforcementAction(alert.GetEnforcement().GetAction()),
			Message: alert.GetEnforcement().GetMessage(),
		},
	}
}

func convertAlertDescription(alert *storage.Alert) string {
	distinct := make(map[string]struct{})
	for _, v := range alert.GetViolations() {
		if vText := v.GetMessage(); vText != "" {
			distinct[v.GetMessage()] = struct{}{}
		}
	}
	distinctSlice := make([]string, 0, len(distinct))
	for v := range distinct {
		distinctSlice = append(distinctSlice, v)
	}
	sort.Strings(distinctSlice)
	return strings.Join(distinctSlice, " ")
}
