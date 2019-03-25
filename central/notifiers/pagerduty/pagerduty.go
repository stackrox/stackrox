package pagerduty

import (
	"bytes"
	"fmt"
	"strings"

	pd "github.com/PagerDuty/go-pagerduty"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	newAlert     = "trigger"
	ackAlert     = "acknowledge"
	resolveAlert = "resolve"
	client       = "StackRox"
)

var (
	log = logging.LoggerForModule()

	severityMap = map[storage.Severity]string{
		storage.Severity_UNSET_SEVERITY:    "info",
		storage.Severity_LOW_SEVERITY:      "warning",
		storage.Severity_MEDIUM_SEVERITY:   "error",
		storage.Severity_HIGH_SEVERITY:     "error",
		storage.Severity_CRITICAL_SEVERITY: "critical",
	}
)

type pagerDuty struct {
	apikey string
	*storage.Notifier
}

func init() {
	notifiers.Add("pagerduty", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := newPagerDuty(notifier)
		return s, err
	})
}

func newPagerDuty(notifier *storage.Notifier) (*pagerDuty, error) {
	pagerDutyConfig, ok := notifier.GetConfig().(*storage.Notifier_Pagerduty)
	if !ok {
		return nil, fmt.Errorf("PagerDuty configuration required")
	}
	conf := pagerDutyConfig.Pagerduty
	if err := validate(conf); err != nil {
		return nil, err
	}
	return &pagerDuty{
		conf.ApiKey,
		notifier}, nil
}

func validate(conf *storage.PagerDuty) error {
	if len(conf.ApiKey) == 0 {
		return fmt.Errorf("PagerDuty API key must be specified")
	}
	return nil
}

func (p *pagerDuty) AlertNotify(alert *storage.Alert) error {
	return p.postAlert(alert, newAlert)
}

func (p *pagerDuty) ProtoNotifier() *storage.Notifier {
	return p.Notifier
}

func (p *pagerDuty) Test() error {
	return p.postAlert(&storage.Alert{
		Id: uuid.NewDummy().String(),
		Policy: &storage.Policy{
			Description: "Test PagerDuty Policy",
			Severity:    storage.Severity_HIGH_SEVERITY,
			Categories:  []string{"Privileges"},
		},
		Deployment: &storage.Deployment{
			Name:        "Test Deployment",
			ClusterName: "Test Cluster",
		},
		Violations: []*storage.Alert_Violation{
			{Message: "This is a sample pagerduty alert message created to test integration with StackRox."},
		},
		Time: types.TimestampNow(),
	}, newAlert)
}

func (p *pagerDuty) AckAlert(alert *storage.Alert) error {
	return p.postAlert(alert, ackAlert)
}

func (p *pagerDuty) ResolveAlert(alert *storage.Alert) error {
	return p.postAlert(alert, resolveAlert)
}

func (p *pagerDuty) postAlert(alert *storage.Alert, eventType string) error {
	pagerDutyEvent, err := p.createPagerDutyEvent(alert, eventType)
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := pd.ManageEvent(pagerDutyEvent)
	if err != nil {
		log.Errorf("PagerDuty response: %+v. Error: %s", resp, err)
	}
	return err
}

// More details on V2 API: https://v2.developer.pagerduty.com/docs/events-api-v2
// PagerDuty has stopped supporting V1 API.
func (p *pagerDuty) createPagerDutyEvent(alert *storage.Alert, eventType string) (pd.V2Event, error) {
	var jsonPayload bytes.Buffer
	err := new(jsonpb.Marshaler).Marshal(&jsonPayload, alert)
	if err != nil {
		return pd.V2Event{}, err
	}

	payload := &pd.V2Payload{
		Summary:   alert.GetPolicy().GetDescription(),
		Source:    fmt.Sprintf("Cluster %s", alert.GetDeployment().GetClusterName()),
		Severity:  severityMap[alert.GetPolicy().GetSeverity()],
		Timestamp: alert.GetTime().String(),
		Class:     strings.Join(alert.GetPolicy().GetCategories(), " "),
		Component: fmt.Sprintf("Deployment %s", alert.GetDeployment().GetName()),
		Details:   jsonPayload,
	}

	return pd.V2Event{
		Action:     eventType,
		RoutingKey: p.apikey,
		Client:     client,
		ClientURL:  notifiers.AlertLink(p.Notifier.UiEndpoint, alert.GetId()),
		DedupKey:   alert.GetId(),
		Payload:    payload,
	}, nil
}
