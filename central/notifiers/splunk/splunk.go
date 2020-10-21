package splunk

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/wrapper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	source                    = "stackrox"
	splunkHECDefaultDataLimit = 10000
	splunkHECHealthEndpoint   = "/services/collector/health/1.0"
	splunkHECEventEndpoint    = "/services/collector/event/1.0"
)

var (
	log = logging.LoggerForModule()

	timeout = 5 * time.Second

	baseURLPattern = regexp.MustCompile(`^(https?://)?[^/]+/*$`)
)

type splunk struct {
	eventEndpoint  string
	healthEndpoint string
	conf           *storage.Splunk

	*storage.Notifier
}

func (s *splunk) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	return s.postAlert(ctx, alert)
}

func (s *splunk) ProtoNotifier() *storage.Notifier {
	return s.Notifier
}

func (s *splunk) Test(ctx context.Context) error {
	if s.healthEndpoint != "" {
		return s.sendHTTPPayload(ctx, http.MethodGet, s.healthEndpoint, nil)
	}
	alert := &storage.Alert{
		Policy:     &storage.Policy{Name: "Test Policy"},
		Deployment: &storage.Alert_Deployment{Name: "Test Deployment"},
		Violations: []*storage.Alert_Violation{
			{Message: "This is a sample Splunk alert message created to test integration with StackRox."},
		},
	}
	return s.postAlert(ctx, alert)
}

func (s *splunk) postAlert(ctx context.Context, alert *storage.Alert) error {
	clonedAlert := alert.Clone()
	// Splunk's HEC by default has a limitation of data size == 10KB
	// Removing some of the fields here to make it smaller
	// More details on HEC limitation: https://developers.perfectomobile.com/display/TT/Splunk+-+Configure+HTTP+Event+Collector
	// Check section on "Increasing the Event Data Truncate Limit"
	notifiers.PruneAlert(clonedAlert, int(s.conf.GetTruncate()))

	return retry.WithRetry(
		func() error {
			return s.sendEvent(ctx, clonedAlert)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (s *splunk) getSplunkEvent(msg proto.Message) (*wrapper.SplunkEvent, error) {
	any, err := protoutils.MarshalAny(msg)
	if err != nil {
		return nil, err
	}
	sourceType := "_json"
	if s.conf.GetDerivedSourceType() {
		_, name := stringutils.Split2(any.GetTypeUrl(), ".")
		sourceType = "stackrox-" + strings.ToLower(strings.Replace(name, ".", "-", -1))
	}

	return &wrapper.SplunkEvent{
		Event:      any,
		Source:     source,
		Sourcetype: sourceType,
	}, nil
}

func (*splunk) Close(ctx context.Context) error {
	return nil
}

func (s *splunk) SendAuditMessage(ctx context.Context, msg *v1.Audit_Message) error {
	if !s.AuditLoggingEnabled() {
		return nil
	}

	return retry.WithRetry(
		func() error {
			return s.sendEvent(ctx, msg)
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

func (s *splunk) sendEvent(ctx context.Context, msg proto.Message) error {
	splunkEvent, err := s.getSplunkEvent(msg)
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

	return s.sendHTTPPayload(ctx, http.MethodPost, s.eventEndpoint, &data)
}

func (s *splunk) sendHTTPPayload(ctx context.Context, method, path string, data io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, path, data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.conf.HttpToken))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.conf.Insecure},
		Proxy:           proxy.FromConfig(),
	}

	client := &http.Client{Transport: tr}
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
	conf := notifier.GetSplunk()
	if conf == nil {
		return nil, errors.New("Splunk configuration required")
	}
	if err := validate(conf); err != nil {
		return nil, err
	}
	url := urlfmt.FormatURL(conf.GetHttpEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	eventEndpoint := url
	var healthEndpoint string
	if baseURLPattern.MatchString(url) {
		eventEndpoint = url + splunkHECEventEndpoint
		healthEndpoint = url + splunkHECHealthEndpoint
	}

	return &splunk{
		conf:           conf,
		eventEndpoint:  eventEndpoint,
		healthEndpoint: healthEndpoint,
		Notifier:       notifier,
	}, nil
}

func validate(conf *storage.Splunk) error {
	if len(conf.HttpToken) == 0 {
		return errors.New("Splunk HTTP Event Collector(HEC) token must be specified")
	}
	if len(conf.HttpEndpoint) == 0 {
		return errors.New("Splunk HTTP endpoint must be specified")
	}
	if conf.GetTruncate() == 0 {
		conf.Truncate = splunkHECDefaultDataLimit
	}
	return nil
}
