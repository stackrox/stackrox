package scanner

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/images/defaults"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite
}

func (s *handlerTestSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
	testutils.SetExampleVersion(s.T())
}

func (s *handlerTestSuite) TestGenerateScannerHTTPHandler() {
	s.T().Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameDevelopmentBuild)
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
