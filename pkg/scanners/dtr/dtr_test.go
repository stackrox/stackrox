package dtr

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/suite"
)

func TestDTRSuite(t *testing.T) {
	suite.Run(t, new(DTRSuite))
}

type DTRSuite struct {
	suite.Suite

	server *httptest.Server
	dtr    types.ImageScanner
}

var statusPayload = `{
  "state": 0,
  "scanner_version": 3,
  "scanner_updated_at": "20171116T21:07:18.934766247Z",
  "db_version": 279,
  "db_updated_at": "20171117T03:14:02.63437292Z",
  "last_db_update_failed": true,
  "replicas": {
   "d8ae913ef3a1": {
    "db_updated_at": "20171116T00:35:27.408476Z",
    "version": "279",
    "replica_id": "d8ae913ef3a1"
   }
  }
 }`

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
		fmt.Fprintf(w, scanResultPayload)
	})
	masterRouter.HandleFunc("/api/v0/imagescan/scan/docker/nginx/1.10/linux/amd64", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "{}")
	})
	masterRouter.HandleFunc("/api/v0/imagescan/status", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, statusPayload)
	})

	masterServer := httptest.NewServer(masterRouter)
	suite.server = masterServer

	protoImageIntegration := &v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Dtr{
			Dtr: &v1.DTRConfig{
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

func (suite *DTRSuite) TestGetScans() {
	d := suite.dtr.(*dtr)

	image := &v1.Image{
		Name: &v1.ImageName{
			Registry: "",
			Remote:   "docker/nginx",
			Tag:      "1.10",
		},
	}
	scans, err := d.GetScans(image)
	suite.NoError(err)

	expectedScanSummaries, err := getExpectedImageScans()
	suite.NoError(err)

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedScans := convertTagScanSummariesToImageScans(d.conf.Endpoint, expectedScanSummaries)
	suite.Equal(expectedScans, scans)
}

func (suite *DTRSuite) TestGetLastScan() {
	d := suite.dtr.(*dtr)

	image := &v1.Image{
		Name: &v1.ImageName{
			Registry: "",
			Remote:   "docker/nginx",
			Tag:      "1.10",
		},
	}
	scan, err := d.GetLastScan(image)
	suite.NoError(err)

	expectedScanSummaries, err := getExpectedImageScans()
	suite.NoError(err)

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedScans := convertTagScanSummariesToImageScans(d.conf.Endpoint, expectedScanSummaries)
	suite.Equal(expectedScans[0], scan)
}

func (suite *DTRSuite) TestScan() {
	d := suite.dtr.(*dtr)
	image := &v1.Image{
		Name: &v1.ImageName{
			Registry: "",
			Remote:   "docker/nginx",
			Tag:      "1.10",
		},
	}
	err := d.Scan(image)
	suite.NoError(err)
}
