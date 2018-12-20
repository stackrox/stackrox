// +build integration

package dtr

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

const (
	dtrServer = "35.185.243.97"
	user      = "srox"
	password  = "f6Ptzm3fUc0cy5HhZ2Rihqpvb5A0Atdv"
)

func TestDTRIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DTRIntegrationSuite))
}

type DTRIntegrationSuite struct {
	suite.Suite

	*dtr
}

func (suite *DTRIntegrationSuite) SetupSuite() {
	integration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Dtr{
			Dtr: &storage.DTRConfig{
				Username: user,
				Password: password,
				Endpoint: dtrServer,
				Insecure: true,
			},
		},
	}

	dtr, err := newScanner(integration)
	suite.NoError(err)

	suite.NoError(dtr.Test())
	suite.dtr = dtr
}

func (suite *DTRIntegrationSuite) TearDownSuite() {}

func (suite *DTRIntegrationSuite) TestGetLastScan() {
	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: dtrServer,
			Remote:   "srox/nginx",
			Tag:      "1.12",
		},
	}
	scan, err := suite.GetLastScan(image)
	suite.Nil(err)
	suite.NotEmpty(scan)
	suite.NotEmpty(scan.GetComponents())
}
