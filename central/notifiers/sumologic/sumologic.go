package generic

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
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

// AlertNotify takes in an alert and generates the Slack message
func (s *sumologic) AlertNotify(alert *storage.Alert) error {
	clonedAlert := protoutils.CloneStorageAlert(alert)
	notifiers.PruneAlert(clonedAlert, 10000)

	var buf bytes.Buffer
	if err := new(jsonpb.Marshaler).Marshal(&buf, clonedAlert); err != nil {
		return err
	}
	return s.sendPayload(&buf)
}

func (s *sumologic) sendPayload(buf io.Reader) error {
	req, err := http.NewRequest(http.MethodPost, s.fullyQualifiedEndpoint, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer utils.IgnoreError(resp.Body.Close)
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Error reading Sumo Logic response body")
		}
		return fmt.Errorf("Received error response from Sumo Logic: %d %s", resp.StatusCode, string(body))
	}
	return nil
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
	fullyQualifiedEndpoint, err := urlfmt.FormatURL(sumoConf.GetHttpSourceAddress(), urlfmt.HTTPS, urlfmt.HonorInputSlash)
	if err != nil {
		return nil, err
	}

	return &sumologic{
		Notifier: notifier,

		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: sumoConf.GetSkipTLSVerify(),
				},
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

func (s *sumologic) Test() error {
	payload := testPayload{
		TestID:      "testalert",
		TestMessage: "This is a test message created to test integration with StackRox.",
	}
	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.sendPayload(bytes.NewBuffer(marshaledPayload))
}

func init() {
	notifiers.Add("sumologic", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		return newSumoLogic(notifier)
	})
}
