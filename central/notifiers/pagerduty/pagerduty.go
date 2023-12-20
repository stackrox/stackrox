package pagerduty

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	pd "github.com/PagerDuty/go-pagerduty"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	imagesTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/utils"
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

	httpStatusCodePattern = regexp.MustCompile(`^HTTP Status Code: ([0-9]{3})\b`)
)

type pagerDuty struct {
	*storage.Notifier
	pdClient   *pd.Client
	routingKey string
}

func newPagerDuty(notifier *storage.Notifier, cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*pagerDuty, error) {
	conf := notifier.GetPagerduty()
	if err := Validate(conf, !env.EncNotifierCreds.BooleanSetting()); err != nil {
		return nil, err
	}

	decCreds := conf.GetApiKey()
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		if notifier.GetNotifierSecret() == "" {
			return nil, errors.Errorf("encrypted notifier credentials for notifier '%s' empty", notifier.GetName())
		}
		decCreds, err = cryptoCodec.Decrypt(cryptoKey, notifier.GetNotifierSecret())
		if err != nil {
			return nil, errors.Errorf("Error decrypting notifier secret for notifier '%s'", notifier.GetName())
		}
	}

	pdClient := pd.NewClient("")
	pdClient.HTTPClient = &http.Client{
		Transport: proxy.RoundTripper(),
	}
	return &pagerDuty{
		Notifier:   notifier,
		pdClient:   pdClient,
		routingKey: decCreds,
	}, nil
}

// Validate PagerDuty notifier
func Validate(conf *storage.PagerDuty, validateSecret bool) error {
	if validateSecret && len(conf.ApiKey) == 0 {
		return errors.New("PagerDuty API key must be specified")
	}
	return nil
}

func (*pagerDuty) Close(context.Context) error {
	return nil
}

func (p *pagerDuty) AlertNotify(_ context.Context, alert *storage.Alert) error {
	return p.postAlert(alert, newAlert)
}

func (p *pagerDuty) ProtoNotifier() *storage.Notifier {
	return p.Notifier
}

func (p *pagerDuty) Test(_ context.Context) error {
	return p.postAlert(&storage.Alert{
		Id: uuid.NewDummy().String(),
		Policy: &storage.Policy{
			Name:        "Test PagerDuty Policy",
			Description: "Sample policy used to test PagerDuty integration",
			Severity:    storage.Severity_HIGH_SEVERITY,
			Categories:  []string{"Privileges"},
		},
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Id:          uuid.NewDummy().String(),
			Name:        "Test Deployment",
			ClusterName: "Test Cluster",
		}},
		Violations: []*storage.Alert_Violation{
			{Message: "This is a sample pagerduty alert message created to test integration with StackRox."},
		},
		Time: types.TimestampNow(),
	}, newAlert)
}

func (p *pagerDuty) AckAlert(_ context.Context, alert *storage.Alert) error {
	return p.postAlert(alert, ackAlert)
}

func (p *pagerDuty) ResolveAlert(_ context.Context, alert *storage.Alert) error {
	return p.postAlert(alert, resolveAlert)
}

func (p *pagerDuty) postAlert(alert *storage.Alert, eventType string) error {
	pagerDutyEvent, err := p.createPagerDutyEvent(alert, eventType)
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := p.pdClient.ManageEvent(&pagerDutyEvent)

	if err != nil {
		log.Errorw("Error sending alert to PagerDuty",
			logging.Any("response", resp), logging.Err(err), logging.ErrCode(codes.PagerDutyGeneric),
			logging.NotifierName(p.GetName()))

		matches := httpStatusCodePattern.FindAllString(err.Error(), 1)
		if len(matches) == 0 {
			return err
		}
		statusCodeStr := strings.TrimSpace(strings.Split(matches[0], ":")[1])
		statusCode, convErr := strconv.Atoi(statusCodeStr)
		if convErr != nil {
			return err
		}
		if statusCode != http.StatusAccepted {
			return errors.Errorf("Received HTTP status code %d from PagerDuty. Check central logs for full error.", statusCode)
		}
	}
	return err
}

// More details on V2 API: https://v2.developer.pagerduty.com/docs/events-api-v2
// PagerDuty has stopped supporting V1 API.
func (p *pagerDuty) createPagerDutyEvent(alert *storage.Alert, eventType string) (pd.V2Event, error) {
	payload := &pd.V2Payload{
		Summary:   notifiers.SummaryForAlert(alert),
		Severity:  severityMap[alert.GetPolicy().GetSeverity()],
		Timestamp: alert.GetTime().String(),
		Class:     strings.Join(alert.GetPolicy().GetCategories(), " "),
		Details:   (*marshalableAlert)(alert),
	}

	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		payload.Source = fmt.Sprintf("%s/%s", entity.Deployment.GetClusterName(), entity.Deployment.GetNamespace())
		payload.Component = fmt.Sprintf("Deployment %s", entity.Deployment.GetName())
	case *storage.Alert_Image:
		payload.Source = fmt.Sprintf("Image from %s/%s", entity.Image.GetName().GetRemote(), entity.Image.GetName().GetRegistry())
		payload.Component = fmt.Sprintf("Image %s", imagesTypes.Wrapper{GenericImage: entity.Image}.FullName())
	case *storage.Alert_Resource_:
		if entity.Resource.GetNamespace() != "" {
			payload.Source = fmt.Sprintf("%s/%s", entity.Resource.GetClusterName(), entity.Resource.GetNamespace())
		} else {
			payload.Source = entity.Resource.GetClusterName()
		}
		payload.Component = fmt.Sprintf("%s %s", entity.Resource.GetResourceType(), entity.Resource.GetName())
	}
	return pd.V2Event{
		Action:     eventType,
		RoutingKey: p.routingKey,
		Client:     client,
		ClientURL:  notifiers.AlertLink(p.Notifier.UiEndpoint, alert),
		DedupKey:   alert.GetId(),
		Payload:    payload,
	}, nil
}

// marshalableAlert type encapsulates the Alert type and adds Marshal method.
type marshalableAlert storage.Alert

// MarshalJSON marshals alert data to bytes, following jsonpb rules.
func (a *marshalableAlert) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := (&jsonpb.Marshaler{}).Marshal(&buf, (*storage.Alert)(a)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshals alert JSON bytes into an Alert object, following jsonpb rules.
func (a *marshalableAlert) UnmarshalJSON(data []byte) error {
	return jsonutil.JSONBytesToProto(data, (*storage.Alert)(a))
}

func init() {
	cryptoKey := ""
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, _, err = notifierUtils.GetActiveNotifierEncryptionKey()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error reading encryption key, notifier will be unable to send notifications"))
		}
	}

	notifiers.Add(notifiers.PagerDutyType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := newPagerDuty(notifier, cryptocodec.Singleton(), cryptoKey)
		return s, err
	})
}
