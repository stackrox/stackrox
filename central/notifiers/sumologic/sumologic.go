package generic

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	timeout = 5 * time.Second
)

// sumologic notifier plugin
type sumologic struct {
	*storage.Notifier

	client                 *http.Client
	fullyQualifiedEndpoint string
}

func (*sumologic) Close(context.Context) error {
	return nil
}

// AlertNotify takes in an alert and generates the Slack message
func (s *sumologic) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	clonedAlert := alert.Clone()
	notifiers.PruneAlert(clonedAlert, 10000)

	return retry.WithRetry(
		func() error {
			return s.sendProtoPayload(ctx, clonedAlert)
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (s *sumologic) sendProtoPayload(ctx context.Context, msg proto.Message) error {
	var buf bytes.Buffer
	if err := new(jsonpb.Marshaler).Marshal(&buf, msg); err != nil {
		return err
	}
	return s.sendPayload(ctx, &buf)
}

func (s *sumologic) sendPayload(ctx context.Context, buf io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.fullyQualifiedEndpoint, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	return notifiers.CreateError(s.GetName(), resp, codes.SumoLogicGeneric)
}

func validateConfig(sumologic *storage.SumoLogic) error {
	errList := errorhelpers.NewErrorList("Sumo Logic notifier validation")
	if sumologic.GetHttpSourceAddress() == "" {
		errList.AddString("http source address is required")
	}
	return errList.ToError()
}

func newSumoLogic(notifier *storage.Notifier) (*sumologic, error) {
	sumoConf := notifier.GetSumologic()
	if err := validateConfig(sumoConf); err != nil {
		return nil, err
	}
	fullyQualifiedEndpoint := urlfmt.FormatURL(sumoConf.GetHttpSourceAddress(), urlfmt.HTTPS, urlfmt.HonorInputSlash)

	return &sumologic{
		Notifier: notifier,

		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: sumoConf.GetSkipTLSVerify(),
				},
				Proxy: proxy.FromConfig(),
			},
		},
		fullyQualifiedEndpoint: fullyQualifiedEndpoint,
	}, nil
}

func (s *sumologic) ProtoNotifier() *storage.Notifier {
	return s.Notifier
}

// Have a separate testPayload struct where the fields
// don't collide with alert fields.
type testPayload struct {
	TestID      string `json:"testID"`
	TestMessage string `json:"testMessage"`
}

func (s *sumologic) Test(ctx context.Context) error {
	payload := testPayload{
		TestID:      "testalert",
		TestMessage: "This is a test message created to test integration with StackRox.",
	}
	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.sendPayload(ctx, bytes.NewBuffer(marshaledPayload))
}

func init() {
	notifiers.Add(notifiers.SumoLogicType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		return newSumoLogic(notifier)
	})
}
