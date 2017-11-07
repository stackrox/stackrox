package dtr

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func handleAuth(r *http.Request) error {
	if r.Header.Get("Authorization") != "Basic dXNlcjpwYXNzd29yZA==" {
		return fmt.Errorf("Not Authorization for request: %v", r.URL.String())
	}
	return nil
}

func (suite *DTRSuite) SetupSuite() {
	masterRouter := http.NewServeMux()
	// Handle
	masterRouter.HandleFunc("/api/v0/imagescan/repositories/docker/nginx/1.10/linux/amd64", func(w http.ResponseWriter, r *http.Request) {
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
		fmt.Fprintf(w, metadataPayload)
	})

	masterServer := httptest.NewServer(masterRouter)
	suite.server = masterServer

	var err error
	// newScanner is tested within setup
	suite.dtr, err = newScanner("http://"+masterServer.Listener.Addr().String(), map[string]string{
		"username": "user",
		"password": "password",
	})
	if err != nil {
		suite.FailNow("Could not setup DTR scanner: " + err.Error())
	}
}

func (suite *DTRSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *DTRSuite) TestGetStatus() {
	d := suite.dtr.(*dtr)
	meta, err := d.getStatus()
	require.Nil(suite.T(), err)

	expectedMeta, err := getExpectedMetadata()
	assert.Equal(suite.T(), expectedMeta, meta)
}

func (suite *DTRSuite) TestGetScans() {
	d := suite.dtr.(*dtr)

	image := &v1.Image{
		Registry: "",
		Remote:   "docker/nginx",
		Tag:      "1.10",
	}
	scans, err := d.GetScans(image)
	assert.Nil(suite.T(), err)

	expectedScanSummaries, err := getExpectedImageScans()
	assert.Nil(suite.T(), err)

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedScans := convertTagScanSummariesToImageScans(d.server, expectedScanSummaries)
	assert.Equal(suite.T(), expectedScans, scans)
}

func (suite *DTRSuite) TestGetLastScan() {
	d := suite.dtr.(*dtr)

	image := &v1.Image{
		Registry: "",
		Remote:   "docker/nginx",
		Tag:      "1.10",
	}
	scan, err := d.GetLastScan(image)
	assert.Nil(suite.T(), err)

	expectedScanSummaries, err := getExpectedImageScans()
	assert.Nil(suite.T(), err)

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedScans := convertTagScanSummariesToImageScans(d.server, expectedScanSummaries)
	assert.Equal(suite.T(), expectedScans[0], scan)
}

func (suite *DTRSuite) TestScan() {
	d := suite.dtr.(*dtr)
	image := &v1.Image{
		Registry: "",
		Remote:   "docker/nginx",
		Tag:      "1.10",
	}
	err := d.Scan(image)
	require.Nil(suite.T(), err)
}
