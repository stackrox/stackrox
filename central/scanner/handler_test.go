package scanner

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/apiparams"
	buildTestutils "github.com/stackrox/stackrox/pkg/buildinfo/testutils"
	testutilsMTLS "github.com/stackrox/stackrox/pkg/mtls/testutils"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stackrox/stackrox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *handlerTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *handlerTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *handlerTestSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.envIsolator)
	s.Require().NoError(err)
	testutils.SetExampleVersion(s.T())
	buildTestutils.SetBuildTimestamp(s.T(), time.Now())
}

func (s *handlerTestSuite) TestGenerateScannerHTTPHandler() {
	server := httptest.NewServer(Handler())
	defer server.Close()

	params := apiparams.Scanner{ClusterType: storage.ClusterType_KUBERNETES_CLUSTER.String(), ScannerImage: "docker.io/stackrox/scanner:latest"}
	marshaledJSON, err := json.Marshal(params)
	s.Require().NoError(err)

	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(marshaledJSON))
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Assert().NotEmpty(body)

	_, err = zip.NewReader(bytes.NewReader(body), int64(len(body)))
	s.Assert().NoError(err)
}
