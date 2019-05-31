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
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/urlfmt"
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
		Deployment: &storage.Deployment{Name: "Test Deployment"},
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
	notifiers.PruneAlert(clonedAlert, int(s.conf.Truncate))
	return s.sendHTTPPayload(clonedAlert)
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
	if !s.GetSplunk().GetAuditLoggingEnabled() {
		return nil
	}
	return s.sendHTTPPayload(msg)
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
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.conf.HttpEndpoint))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.conf.Insecure},
	}

	client := &http.Client{Timeout: timeout, Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, err := httputil.ReadResponse(resp)
		if err != nil {
			return fmt.Errorf("HTTP Status Code: %d", resp.StatusCode)
		}
		return fmt.Errorf("HTTP Status Code: %d - %s", resp.StatusCode, string(body))
	}
	return nil
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
