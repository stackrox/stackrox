// +build integration

package clair

import (
	"testing" // This is needed to register Docker registries.

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/registries"
	_ "github.com/stackrox/rox/pkg/registries/docker"
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
	protoImageIntegration := &v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Clair{
			Clair: &v1.ClairConfig{
				Endpoint: "http://localhost:6060",
			},
		},
	}

	c, err := newScanner(protoImageIntegration)
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
		Name: &v1.ImageName{
			Remote: "library/nginx",
			Tag:    "1.13",
		},
	}

	creator := registries.Registry["docker"]
	s, err := creator(&v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Clair{
			Clair: &v1.ClairConfig{
				Endpoint: "registry-1.docker.io",
			},
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
