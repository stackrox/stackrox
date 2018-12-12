package cscc

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/cscc/client"
	"github.com/stackrox/rox/pkg/notifiers/cscc/findings"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	logger = logging.LoggerForModule()
)

// The CSCC notifier plugin integrates with Google's Cloud Security Command Center.
type cscc struct {
	// The Service Account is a Google JSON service account key.
	// The GCP Organization ID is a numeric identifier for the Google Cloud Platform
	// organization. It is required so that we can tag findings to the right org.
	client client.Config
	// The GCP Project is a string identifier for the Google Cloud Platform
	// project. It is required so that we can tag findings to the right org.
	// This can be inferred from service accounts but in alpha testing we are
	// using a key from a separate project.
	gcpProject string

	*v1.Notifier
}

type config struct {
	ServiceAccount string `json:"serviceAccount"`
	GCPOrgID       string `json:"gcpOrgID"`
	GCPProject     string `json:"gcpProject"`
}

func (c config) validate() error {
	if c.ServiceAccount == "" {
		return errors.New("serviceAccount must be defined in the CSCC Configuration")
	}
	if c.GCPOrgID == "" {
		return errors.New("gcpOrgID must be defined in the CSCC Configuration")
	}
	if err := client.ValidateOrgID(c.GCPOrgID); err != nil {
		return err
	}
	if c.GCPProject == "" {
		return errors.New("gcpProject must be defined in the CSCC Configuration")
	}
	return nil
}

func (c *cscc) getAlertDescription(alert *v1.Alert) string {
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

func transformSeverity(s storage.Severity) string {
	switch s {
	case storage.Severity_LOW_SEVERITY:
		return "low"
	case storage.Severity_MEDIUM_SEVERITY:
		return "medium"
	case storage.Severity_HIGH_SEVERITY:
		return "high"
	case storage.Severity_CRITICAL_SEVERITY:
		return "critical"
	default:
		return "info"
	}
}

func transformEnforcement(a storage.EnforcementAction) string {
	switch a {
	case storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:
		return "Scaled to zero replicas"
	case storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT:
		return "Unsatisfiable node constraint added to prevent deployment"
	default:
		return a.String()
	}
}

func alertEnforcement(alert *v1.Alert) []findings.Enforcement {
	if alert.GetEnforcement().GetAction() == storage.EnforcementAction_UNSET_ENFORCEMENT {
		return nil
	}
	return []findings.Enforcement{
		{
			Action:  transformEnforcement(alert.GetEnforcement().GetAction()),
			Message: alert.GetEnforcement().GetMessage(),
		},
	}
}

func (c *cscc) NetworkPolicyYAMLNotify(yaml string, clusterName string) error {
	// We will not bubble up the information that yaml notifications were sent out in
	// cscc interface, so do nothing
	return nil
}

//AlertNotify takes in an alert and generates the notification
func (c *cscc) AlertNotify(alert *v1.Alert) error {
	alertLink := notifiers.AlertLink(c.Notifier.UiEndpoint, alert.GetId())
	summary := c.getAlertDescription(alert)

	category := alert.GetPolicy().GetName()
	severity := transformSeverity(alert.GetPolicy().GetSeverity())
	finding := &findings.SourceFinding{
		ID:       alert.GetId(),
		Category: category,
		AssetIDs: []string{findings.ClusterID{
			Org:     c.client.GCPOrganizationID,
			Project: c.gcpProject,
			ID:      alert.GetDeployment().GetClusterName(),
		}.AssetID()},
		SourceID:  findings.SourceID,
		Timestamp: protoconv.ConvertGoGoProtoTimeToGolangProtoTime(alert.GetTime()),
		URL:       alertLink,
		Properties: findings.Properties{
			SCCCategory:     category,
			SCCStatus:       "active",
			SCCSeverity:     severity,
			SCCSourceStatus: "active",

			Namespace:      alert.GetDeployment().GetNamespace(),
			Service:        alert.GetDeployment().GetName(),
			DeploymentType: alert.GetDeployment().GetType(),
			// Container info is not available for Prevent.

			EnforcementActions: alertEnforcement(alert),
			Summary:            summary,
		}.Map(),
	}

	return c.client.CreateFinding(finding)
}

// BenchmarkNotify does nothing currently, since we do not want to post
// benchmarks to CSCC.
func (c *cscc) BenchmarkNotify(schedule *v1.BenchmarkSchedule) error {
	return nil
}

func newCSCC(protoNotifier *v1.Notifier) (*cscc, error) {
	csccConfig, ok := protoNotifier.GetConfig().(*v1.Notifier_Cscc)
	if !ok {
		return nil, fmt.Errorf("CSCC config is required")
	}
	conf := csccConfig.Cscc

	cfg := &config{
		ServiceAccount: conf.ServiceAccount,
		GCPOrgID:       conf.GcpOrgId,
		GCPProject:     conf.GcpProject,
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return newWithConfig(protoNotifier, cfg), nil
}

func newWithConfig(protoNotifier *v1.Notifier, cfg *config) *cscc {
	return &cscc{
		Notifier: protoNotifier,
		client: client.Config{
			ServiceAccount:    []byte(cfg.ServiceAccount),
			GCPOrganizationID: cfg.GCPOrgID,
			Logger:            logger,
		},
		gcpProject: cfg.GCPProject,
	}
}

func (c *cscc) ProtoNotifier() *v1.Notifier {
	return c.Notifier
}

func (c *cscc) Test() error {
	return errors.New("Test is not yet implemented for CSCC")
}

func init() {
	notifiers.Add("cscc", func(notifier *v1.Notifier) (notifiers.Notifier, error) {
		j, err := newCSCC(notifier)
		return j, err
	})
}
