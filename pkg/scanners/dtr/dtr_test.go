package dtr

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/suite"
)

const emptyScan = `
[
  {
   "namespace": "qa",
   "reponame": "socat",
   "tag": "testing",
   "critical": 0,
   "major": 0,
   "minor": 0,
   "last_scan_status": 0,
   "check_completed_at": "0001-01-01T00:00:00Z",
   "should_rescan": false,
   "has_foreign_layers": false
  }
 ]
`

func TestDTRSuite(t *testing.T) {
	suite.Run(t, new(DTRSuite))
}

type DTRSuite struct {
	suite.Suite

	server *httptest.Server
	dtr    types.ImageScanner
}

func handleAuth(r *http.Request) error {
	if r.Header.Get("Authorization") != "Basic dXNlcjpwYXNzd29yZA==" {
		return fmt.Errorf("Not Authorization for request: %v", r.URL.String())
	}
	return nil
}

func (suite *DTRSuite) SetupSuite() {
	masterRouter := http.NewServeMux()
	// Handle
	masterRouter.HandleFunc("/api/v0/imagescan/repositories/docker/nginx/1.10", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, scanResultPayload)
		suite.NoError(err)
	})
	masterRouter.HandleFunc("/api/v0/imagescan/repositories/docker/nginx/1.11", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, emptyScan)
		suite.NoError(err)
	})

	masterServer := httptest.NewServer(masterRouter)
	suite.server = masterServer

	protoImageIntegration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Dtr{
			Dtr: &storage.DTRConfig{
				Username: "user",
				Password: "password",
				Endpoint: "http://" + masterServer.Listener.Addr().String(),
			},
		},
	}

	var err error
	// newScanner is tested within setup
	suite.dtr, err = newScanner(protoImageIntegration)
	if err != nil {
		suite.FailNow("Could not setup DTR scanner: " + err.Error())
	}
}

func (suite *DTRSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *DTRSuite) TestTestFunc() {
	d := suite.dtr.(*dtr)
	suite.NoError(d.Test())
}

func (suite *DTRSuite) TestGetLastScan() {
	d := suite.dtr.(*dtr)

	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "",
			Remote:   "docker/nginx",
			Tag:      "1.10",
		},
	}
	scan, err := d.GetLastScan(image)
	suite.NoError(err)

	expectedScanSummary, err := getExpectedImageScan()
	suite.NoError(err)

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedScan := convertTagScanSummaryToImageScan(image, expectedScanSummary)
	suite.Equal(expectedScan, scan)
}

func (suite *DTRSuite) TestGetLastScanWithEmptyResult() {
	d := suite.dtr.(*dtr)

	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "",
			Remote:   "docker/nginx",
			Tag:      "1.11",
		},
	}
	_, err := d.GetLastScan(image)
	suite.Error(err)
}
