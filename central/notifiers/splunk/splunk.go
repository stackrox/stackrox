package splunk

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/wrapper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	source                    = "stackrox"
	sourceType                = "_json"
	splunkHECDefaultDataLimit = 10000
)

var (
	log = logging.LoggerForModule()

	timeout = 5 * time.Second
)

type splunk struct {
	endpoint string
	conf     *storage.Splunk

	*storage.Notifier
}

func (s *splunk) AlertNotify(alert *storage.Alert) error {
	return s.postAlert(alert)
}

func (s *splunk) ProtoNotifier() *storage.Notifier {
	return s.Notifier
}

func (s *splunk) Test() error {
	alert := &storage.Alert{
		Policy:     &storage.Policy{Name: "Test Policy"},
		Deployment: &storage.Alert_Deployment{Name: "Test Deployment"},
		Violations: []*storage.Alert_Violation{
			{Message: "This is a sample Splunk alert message created to test integration with StackRox."},
		},
	}
	return s.postAlert(alert)
}

func (s *splunk) postAlert(alert *storage.Alert) error {
	clonedAlert := protoutils.CloneStorageAlert(alert)
	// Splunk's HEC by default has a limitation of data size == 10KB
	// Removing some of the fields here to make it smaller
	// More details on HEC limitation: https://developers.perfectomobile.com/display/TT/Splunk+-+Configure+HTTP+Event+Collector
	// Check section on "Increasing the Event Data Truncate Limit"
	notifiers.PruneAlert(clonedAlert, int(s.conf.GetTruncate()))

	return retry.WithRetry(
		func() error {
			return s.sendHTTPPayload(clonedAlert)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func getSplunkEvent(msg proto.Message) (*wrapper.SplunkEvent, error) {
	any, err := protoutils.MarshalAny(msg)
	if err != nil {
		return nil, err
	}
	return &wrapper.SplunkEvent{
		Event:      any,
		Source:     source,
		Sourcetype: sourceType,
	}, nil
}

func (s *splunk) SendAuditMessage(msg *v1.Audit_Message) error {
	if !s.AuditLoggingEnabled() {
		return nil
	}

	return retry.WithRetry(
		func() error {
			return s.sendHTTPPayload(msg)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (s *splunk) AuditLoggingEnabled() bool {
	return s.GetSplunk().GetAuditLoggingEnabled()
}

func (s *splunk) sendHTTPPayload(msg proto.Message) error {
	splunkEvent, err := getSplunkEvent(msg)
	if err != nil {
		return err
	}

	var data bytes.Buffer
	err = new(jsonpb.Marshaler).Marshal(&data, splunkEvent)
	if err != nil {
		return err
	}

	if data.Len() > int(s.conf.GetTruncate()) {
		return fmt.Errorf("Splunk HEC truncate data limit (%d bytes) exceeded: %d", s.conf.GetTruncate(), data.Len())
	}

	req, err := http.NewRequest(http.MethodPost, s.endpoint, &data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.conf.HttpToken))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.conf.Insecure},
	}

	client := &http.Client{Timeout: timeout, Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	return notifiers.CreateError("Splunk", resp)
}

func init() {
	notifiers.Add("splunk", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := newSplunk(notifier)
		return s, err
	})
}

func newSplunk(notifier *storage.Notifier) (*splunk, error) {
	splunkConfig, ok := notifier.GetConfig().(*storage.Notifier_Splunk)
	if !ok {
		return nil, fmt.Errorf("Splunk configuration required")
	}
	conf := splunkConfig.Splunk
	if err := validate(conf); err != nil {
		return nil, err
	}
	endpoint, err := urlfmt.FormatURL(conf.GetHttpEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}

	return &splunk{
		conf:     conf,
		endpoint: endpoint,
		Notifier: notifier,
	}, nil
}

func validate(conf *storage.Splunk) error {
	if len(conf.HttpToken) == 0 {
		return fmt.Errorf("Splunk HTTP Event Collector(HEC) token must be specified")
	}
	if len(conf.HttpEndpoint) == 0 {
		return fmt.Errorf("Splunk HTTP endpoint must be specified")
	}
	if conf.GetTruncate() == 0 {
		conf.Truncate = splunkHECDefaultDataLimit
	}
	return nil
}
