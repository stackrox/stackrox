// +build integration

package dtr

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

const dtrServer = "https://mitigate-dtr.rox.systems"
const user = "srox"
const password = "f6Ptzm3fUc0cy5HhZ2Rihqpvb5A0Atdv"

func TestDTRIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DTRIntegrationSuite))
}

type DTRIntegrationSuite struct {
	suite.Suite

	*dtr
}

func (suite *DTRIntegrationSuite) SetupSuite() {
	dtr := &dtr{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		server:   dtrServer,
		username: user,
		password: password,
	}

	err := dtr.fetchMetadata()
	suite.NoError(err)
	suite.dtr = dtr
}

func (suite *DTRIntegrationSuite) TearDownSuite() {}

func (suite *DTRIntegrationSuite) TestGetStatus() {
	scannerStatus, features, err := suite.getStatus()
	suite.Nil(err)
	suite.NotNil(scannerStatus)
	suite.NotEqual(scannerStatus.DBVersion, 0)
	suite.NotNil(features)
	suite.True(features.ScanningEnabled)
}

func (suite *DTRIntegrationSuite) TestGetScans() {
	image := &v1.Image{
		Registry: dtrServer,
		Remote:   "srox/nginx",
		Tag:      "1.10",
	}
	scans, err := suite.GetScans(image)
	suite.Nil(err)
	suite.NotEmpty(scans)
	suite.NotEmpty(scans[0].Components)
}

func (suite *DTRIntegrationSuite) TestGetLastScan() {
	image := &v1.Image{
		Registry: dtrServer,
		Remote:   "srox/nginx",
		Tag:      "1.10",
	}
	scan, err := suite.GetLastScan(image)
	suite.Nil(err)
	suite.NotNil(scan)
	suite.NotEmpty(scan.Components)
}

func (suite *DTRIntegrationSuite) TestScan() {
	image := &v1.Image{
		Registry: "",
		Remote:   "srox/nginx",
		Tag:      "1.10",
	}
	err := suite.Scan(image)
	suite.Nil(err)
}
