// +build integration

package quay

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

const (
	testOauthToken = "0j9dhT9jCNFpsVAzwLavnyeEy2HWnrfTQnbJgQF8"
)

func TestQuayIntegrationSuite(t *testing.T) {
	suite.Run(t, new(QuayIntegrationSuite))
}

type QuayIntegrationSuite struct {
	suite.Suite

	quay *quay
}

func (suite *QuayIntegrationSuite) SetupSuite() {
	protoScanner := &v1.Scanner{
		Endpoint:   "quay.io",
		Registries: []string{"quay.io"},
		Config: map[string]string{
			"oauthToken": testOauthToken,
		},
	}

	q, err := newScanner(protoScanner)
	suite.NoError(err)
	suite.quay = q
}

func (suite *QuayIntegrationSuite) TearDownSuite() {}

func (suite *QuayIntegrationSuite) TestScanTest() {
	err := suite.quay.Test()
	suite.NoError(err)
}

func (suite *QuayIntegrationSuite) TestGetLastScan() {
	image := &v1.Image{
		Name: &v1.ImageName{
			Sha:      "d088ff453bb180ade5c97c8e7961afbbb6921f0131982563de431e8d3d9bb606",
			Registry: "quay.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}
	scan, err := suite.quay.GetLastScan(image)
	suite.Nil(err)
	suite.NotNil(scan)
	if scan != nil {
		suite.NotEmpty(scan.Components)
	}
}
