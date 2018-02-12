// +build integration

package clair

import (
	"testing"

	_ "bitbucket.org/stack-rox/apollo/pkg/registries/docker"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"github.com/stretchr/testify/suite"
)

func TestClairIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ClairIntegrationSuite))
}

type ClairIntegrationSuite struct {
	suite.Suite

	clair *clair
}

func (suite *ClairIntegrationSuite) SetupSuite() {
	protoScanner := &v1.Scanner{
		Endpoint: "http://localhost:6060",
	}

	c, err := newScanner(protoScanner)
	suite.NoError(err)
	suite.clair = c
}

func (suite *ClairIntegrationSuite) TearDownSuite() {}

func (suite *ClairIntegrationSuite) TestScanTest() {
	err := suite.clair.Test()
	suite.NoError(err)
}

func (suite *ClairIntegrationSuite) TestGetLastScan() {
	image := &v1.Image{
		Remote: "library/nginx",
		Tag:    "1.13",
	}

	creator := registries.Registry["docker"]
	s, err := creator(&v1.Registry{
		Endpoint:      "registry-1.docker.io",
		ImageRegistry: "docker.io",
		Config: map[string]string{
			"username": "",
			"password": "",
		},
	})
	if err != nil {
		panic(err)
	}
	metadata, err := s.Metadata(image)
	if err != nil {
		panic(err)
	}
	image.Metadata = metadata

	scan, err := suite.clair.GetLastScan(image)
	suite.Nil(err)
	suite.NotNil(scan)
	if scan != nil {
		suite.NotEmpty(scan.Components)
	}
}
