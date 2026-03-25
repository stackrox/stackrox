//go:build sql_integration

package splunk

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	violationTime      = time.Date(2024, time.June, 14, 20, 00, 00, 0, time.UTC)
	violationTimestamp = protocompat.ConvertTimeToTimestampOrNil(&violationTime)

	//go:embed testdata/generic_violation.json
	expectedGenericViolation string

	//go:embed testdata/file_access_violation.json
	expectedFileAccessViolation string
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

func (s *violationSerializationTestSuite) queryViolations(alert *storage.Alert) string {
	ctx := sac.WithAllAccess(context.Background())

	err := s.alertStore.UpsertAlert(ctx, alert)
	s.Require().NoError(err)

	handler := newViolationsHandler(s.alertStore, defaultPaginationSettings)
	server := httptest.NewServer(wrapHandler(ctx, handler))
	defer server.Close()

	client := server.Client()
	client.Timeout = 5 * time.Second
	requestBody := bytes.NewBufferString("")
	queryString := "?from_checkpoint=2000-01-01T00:00:00Z__2024-06-26T22:00:00Z"
	req, reqErr := http.NewRequest(http.MethodPost, server.URL+queryString, requestBody)
	s.Require().NoError(reqErr)
	resp, err := client.Do(req)
	s.Require().NoError(err)

	defer func() { s.NoError(resp.Body.Close()) }()

	respBodyData, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	return string(respBodyData)
}

func (s *violationSerializationTestSuite) TestViolationSerialization() {
	alert := fixtures.GetSerializationTestAlert()
	alert.Time = violationTimestamp

	s.JSONEq(expectedGenericViolation, s.queryViolations(alert))
}

func (s *violationSerializationTestSuite) TestFileAccessViolationSerialization() {
	alert := &storage.Alert{
		Id: "fa1eacce-0000-4000-a000-000000000001",
		Policy: &storage.Policy{
			Id:              "fa1eacce-0000-4000-b000-000000000002",
			Name:            "File Access: /etc/passwd",
			Description:     "Detect modifications to /etc/passwd",
			Severity:        storage.Severity_HIGH_SEVERITY,
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			PolicyVersion:   "1.1",
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:        "fa1eacce-0000-4000-c000-000000000003",
				Name:      "nginx",
				Type:      "Deployment",
				Namespace: "default",
				ClusterId: "fa1eacce-0000-4000-d000-000000000004",
			},
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "'/etc/passwd' opened (writable)",
				MessageAttributes: &storage.Alert_Violation_FileAccess{
					FileAccess: &storage.FileAccess{
						File: &storage.FileAccess_File{
							EffectivePath: "/etc/passwd",
							ActualPath:    "/rootfs/etc/passwd",
							Meta: &storage.FileAccess_FileMetadata{
								Uid:      0,
								Gid:      0,
								Mode:     0644,
								Username: "root",
								Group:    "root",
							},
						},
						Operation: storage.FileAccess_OPEN,
						Timestamp: violationTimestamp,
						Process: &storage.ProcessIndicator{
							Id:            "fa1eacce-0000-4000-f000-000000000006",
							DeploymentId:  "fa1eacce-0000-4000-c000-000000000003",
							ContainerName: "nginx",
							PodId:         "nginx-pod",
							Signal: &storage.ProcessSignal{
								Id:           "fa1eacce-0000-4000-f000-000000000007",
								Name:         "vi",
								Args:         "/etc/passwd",
								ExecFilePath: "/usr/bin/vi",
								Pid:          42,
								Uid:          0,
								Gid:          0,
								Time:         violationTimestamp,
							},
						},
						Hostname: "node-1",
					},
				},
				Type: storage.Alert_Violation_FILE_ACCESS,
			},
		},
		Time: violationTimestamp,
	}

	s.JSONEq(expectedFileAccessViolation, s.queryViolations(alert))
}

func wrapHandler(ctx context.Context, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrappedRequest := r.Clone(ctx)
		handler(w, wrappedRequest)
	}
}
