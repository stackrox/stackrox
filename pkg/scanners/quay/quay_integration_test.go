// +build integration

package quay

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
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
	protoImageIntegration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Quay{
			Quay: &storage.QuayConfig{
				OauthToken: testOauthToken,
				Endpoint:   "quay.io",
			},
		},
	}

	q, err := newScanner(protoImageIntegration)
	suite.NoError(err)
	suite.NoError(q.Test())
	suite.quay = q
}

func (suite *QuayIntegrationSuite) TearDownSuite() {}

func (suite *QuayIntegrationSuite) TestScanTest() {
	err := suite.quay.Test()
	suite.NoError(err)
}

func (suite *QuayIntegrationSuite) TestGetLastScan() {
	image := &storage.Image{
		Id: "sha256:d088ff453bb180ade5c97c8e7961afbbb6921f0131982563de431e8d3d9bb606",
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}

	var scan *storage.ImageScan
	err := retry.WithRetry(func() error {
		var err error
		scan, err = suite.quay.GetLastScan(image)
		return err
	},
		retry.Tries(5),
		retry.BetweenAttempts(func() {
			time.Sleep(time.Second)
		}),
	)
	suite.NoError(err)
	suite.NotEmpty(scan.GetComponents())
}
