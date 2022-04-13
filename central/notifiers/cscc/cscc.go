package cscc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/notifiers"
	"github.com/stackrox/stackrox/central/notifiers/cscc/client"
	"github.com/stackrox/stackrox/central/notifiers/cscc/findings"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/protoconv"
	"github.com/stackrox/stackrox/pkg/retry"
	"github.com/stackrox/stackrox/pkg/sac"
)

var (
	log = logging.LoggerForModule()

	clusterForAlertContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster),
		))
)

func init() {
	notifiers.Add("cscc", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		j, err := newCSCC(notifier)
		return j, err
	})
}

// The Cloud SCC notifier plugin integrates with Google's Cloud Security Command Center.
type cscc struct {
	// The Service Account is a Google JSON service account key.
	// The GCP Organization ID is a numeric identifier for the Google Cloud Platform
	// organization. It is required so that we can tag findings to the right org.
	client client.Config
	config *config

	*storage.Notifier
}

type config struct {
	ServiceAccount string `json:"serviceAccount"`
	SourceID       string `json:"sourceID"`
}

func (c config) validate() error {
	if c.ServiceAccount == "" {
		return errors.New("serviceAccount must be defined in the Cloud SCC Configuration")
	}
	if c.SourceID == "" {
		return errors.New("sourceID must be defined in the Cloud SCC Configuration")
	}
	if err := client.ValidateSourceID(c.SourceID); err != nil {
		return err
	}
	return nil
}

func (c *cscc) getAlertDescription(alert *storage.Alert) string {
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

func alertEnforcement(alert *storage.Alert) []findings.Enforcement {
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

// Cloud SCC requires that Finding IDs be alphanumeric (no special characters)
// and 1-32 characters long. UUIDs are 32 characters if you remove hyphens.
func processUUID(u string) string {
	return strings.Replace(u, "-", "", -1)
}

func (c *cscc) getCluster(id string) (*storage.Cluster, error) {
	cluster, exists, err := clusterDatastore.Singleton().GetCluster(clusterForAlertContext, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Could not retrieve cluster %q because it does not exist", id)
	}
	providerMetadata := cluster.GetStatus().GetProviderMetadata()
	if providerMetadata.GetGoogle().GetProject() == "" {
		return nil, fmt.Errorf("Could not find Google project for cluster %q", id)
	}
	if providerMetadata.GetGoogle().GetClusterName() == "" {
		return nil, fmt.Errorf("Could not find Google cluster name for cluster %q", id)
	}
	if providerMetadata.GetZone() == "" {
		return nil, fmt.Errorf("Could not find Google zone for cluster %q", id)
	}
	return cluster, nil
}

func (c *cscc) Close(ctx context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the notification
func (c *cscc) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	if alert.GetImage() != nil {
		return errors.New("CSCC integration can only handle alerts for deployments and resources")
	}

	alertLink := notifiers.AlertLink(c.Notifier.UiEndpoint, alert)
	summary := c.getAlertDescription(alert)

	findingID := processUUID(alert.GetId())

	cluster, err := c.getCluster(alert.GetDeployment().GetClusterId())
	if err != nil {
		return err
	}
	providerMetadata := cluster.GetStatus().GetProviderMetadata()

	category := alert.GetPolicy().GetName()
	severity := transformSeverity(alert.GetPolicy().GetSeverity())
	finding := &findings.Finding{
		ID:     fmt.Sprintf("%s/findings/%s", c.config.SourceID, findingID),
		Parent: c.config.SourceID,
		ResourceName: findings.ClusterID{
			Project: providerMetadata.GetGoogle().GetProject(),
			Zone:    providerMetadata.GetZone(),
			Name:    providerMetadata.GetGoogle().GetClusterName(),
		}.ResourceName(),
		Category:  category,
		URL:       alertLink,
		Timestamp: protoconv.ConvertTimestampToTimeOrNow(alert.GetTime()).Format(time.RFC3339Nano),
	}

	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		finding.Properties = findings.Properties{
			Severity: severity,

			Namespace:      alert.GetDeployment().GetNamespace(),
			Service:        alert.GetDeployment().GetName(),
			DeploymentType: alert.GetDeployment().GetType(),

			EnforcementActions: alertEnforcement(alert),
			Summary:            summary,
		}.Map()
	case *storage.Alert_Resource_:
		findings.Properties{
			Severity: severity,

			Namespace:    alert.GetResource().GetNamespace(),
			Service:      alert.GetResource().GetName(),
			ResourceType: alert.GetResource().GetResourceType().String(),

			EnforcementActions: alertEnforcement(alert),
			Summary:            summary,
		}.Map()
	}

	if alert.GetState() == storage.ViolationState_ATTEMPTED {
		finding.State = findings.StateInactive
	} else {
		finding.State = findings.StateActive
	}

	return retry.WithRetry(
		func() error {
			return c.client.CreateFinding(ctx, finding, findingID)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func newCSCC(protoNotifier *storage.Notifier) (*cscc, error) {
	csccConfig, ok := protoNotifier.GetConfig().(*storage.Notifier_Cscc)
	if !ok {
		return nil, errors.New("Cloud SCC config is required")
	}
	conf := csccConfig.Cscc

	cfg := &config{
		ServiceAccount: conf.ServiceAccount,
		SourceID:       conf.SourceId,
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return newWithConfig(protoNotifier, cfg), nil
}

func newWithConfig(protoNotifier *storage.Notifier, cfg *config) *cscc {
	return &cscc{
		Notifier: protoNotifier,
		client: client.Config{
			ServiceAccount: []byte(cfg.ServiceAccount),
			SourceID:       cfg.SourceID,
			Logger:         log,
		},
		config: cfg,
	}
}

func (c *cscc) ProtoNotifier() *storage.Notifier {
	return c.Notifier
}

func (c *cscc) Test(context.Context) error {
	return errors.New("Test is not yet implemented for Cloud SCC")
}
