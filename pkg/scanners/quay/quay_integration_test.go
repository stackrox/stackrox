//go:build integration

package quay

import (
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/suite"
)

const (
	testOauthToken = "0j9dhT9jCNFpsVAzwLavnyeEy2HWnrfTQnbJgQF8" //#nosec G101
)

func TestQuayIntegrationSuite(t *testing.T) {
	t.Skip("See ROX-9448 for re-enabling")
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
	suite.NoError(filterOkErrors(q.Test()))
	suite.quay = q
}

func (suite *QuayIntegrationSuite) TearDownSuite() {}

func (suite *QuayIntegrationSuite) TestScanTest() {
	err := suite.quay.Test()
	suite.NoError(filterOkErrors(err))
}

func (suite *QuayIntegrationSuite) TestGetScan() {
	image := &storage.Image{
		Id: "sha256:d088ff453bb180ade5c97c8e7961afbbb6921f0131982563de431e8d3d9bb606",
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}

	var scan *storage.ImageScan
	var err error
	err = retry.WithRetry(func() error {
		scan, err = suite.quay.GetScan(image)
		err = filterOkErrors(err)
		if err != nil {
			return retry.MakeRetryable(err)
		}
		return nil
	}, retry.OnFailedAttempts(func(err error) {
		suite.T().Logf("error scanning image: %v", err)
		time.Sleep(5 * time.Second)
	}), retry.Tries(10))

	suite.NoError(err)
	suite.NotEmpty(scan.GetComponents())
}

func filterOkErrors(err error) error {
	if err != nil &&
		(strings.Contains(err.Error(), "EOF") ||
			strings.Contains(err.Error(), "status=502")) {
		// Ignore failures that can indicate quay.io outage
		return nil
	}
	return err
}
