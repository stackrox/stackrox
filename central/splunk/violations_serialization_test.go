//go:build sql_integration

package splunk

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	violationTime      = time.Date(2024, time.June, 14, 20, 00, 00, 0, time.UTC)
	violationTimestamp = protocompat.ConvertTimeToTimestampOrNil(&violationTime)
)

func TestViolationSerialization(t *testing.T) {
	suite.Run(t, new(violationSerializationTestSuite))
}

type violationSerializationTestSuite struct {
	suite.Suite

	pool       *pgtest.TestPostgres
	alertStore alertDataStore.DataStore
}

func (s *violationSerializationTestSuite) SetupTest() {
	s.pool = pgtest.ForT(s.T())
	alertStore := alertDataStore.GetTestPostgresDataStore(s.T(), s.pool)
	s.alertStore = alertStore
}

func (s *violationSerializationTestSuite) TestViolationSerialization() {
	ctx := sac.WithAllAccess(context.Background())

	// Inject alert
	alert := fixtures.GetSerializationTestAlert()
	alert.Time = violationTimestamp
	err := s.alertStore.UpsertAlert(ctx, alert)
	s.Require().NoError(err)

	// Serve with the target function
	handler := newViolationsHandler(s.alertStore, defaultPaginationSettings)
	server := httptest.NewServer(wrapHandler(ctx, handler))
	defer server.Close()

	// Query server
	client := server.Client()
	client.Timeout = 5 * time.Second
	requestBody := bytes.NewBufferString("")
	queryString := "?from_checkpoint=2000-01-01T00:00:00Z__2024-06-26T22:00:00Z"
	req, reqErr := http.NewRequest(http.MethodPost, server.URL+queryString, requestBody)
	s.Require().NoError(reqErr)
	resp, err := client.Do(req)
	s.Require().NoError(err)

	// Validate response
	respBody := resp.Body
	defer func() { s.NoError(respBody.Close()) }()

	expectedViolationResponse := `{
	"newCheckpoint": "2024-06-26T22:00:00Z",
	"violations": [
		{
			"alertInfo": {
				"alertId": "aeaaaaaa-bbbb-4011-0000-111111111111"
			},
			"deploymentInfo": {},
			"violationInfo": {
				"containerName": "nginx",
				"podId": "nginx",
				"violationId": "f073c9f5-c766-5dc4-b7cb-aa8ac8f2d445",
				"violationMessage": "This is a kube event violation",
				"violationMessageAttributes": [
					{
						"key": "pod",
						"value": "nginx"
					},
					{
						"key": "container",
						"value": "nginx"
					}
				],
				"violationTime": "2024-06-14T20:00:00Z",
				"violationType": "GENERIC"
			}
		},
		{
			"alertInfo": {
				"alertId": "aeaaaaaa-bbbb-4011-0000-111111111111"
			},
			"deploymentInfo": {},
			"violationInfo": {
				"violationId": "aeaaaaaa-bbbb-4011-0000-111111111111",
				"violationMessage": "Deployment is affected by 'CVE-2017-15670'",
				"violationTime": "2024-06-14T20:00:00Z",
				"violationType": "GENERIC"
			}
		}
	]
}`

	respBodyData, err := io.ReadAll(respBody)
	s.NoError(err)
	s.JSONEq(expectedViolationResponse, string(respBodyData))
}

func wrapHandler(ctx context.Context, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrappedRequest := r.Clone(ctx)
		handler(w, wrappedRequest)
	}
}
