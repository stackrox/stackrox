// +build integration

package tenable

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

const (
	accessKey = "54d75bd30474079b62761b5913917c27a8bb8f781b823c2d8d51dda687180bf3"
	secretKey = "0dbf0fe9bf34117ca49b40cf36eab72c9e2cb2247739dcbd2706fdf9cc4cb0e3"
)

func TestTenableIntegrationSuite(t *testing.T) {
	suite.Run(t, new(TenableIntegrationSuite))
}

type TenableIntegrationSuite struct {
	suite.Suite

	tenable *tenable
}

func (suite *TenableIntegrationSuite) SetupSuite() {
	tenable := &tenable{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		accessKey: accessKey,
		secretKey: secretKey,
	}
	suite.tenable = tenable
}

func (suite *TenableIntegrationSuite) TearDownSuite() {}

func (suite *TenableIntegrationSuite) TestTestFunction() {
	err := suite.tenable.Test()
	suite.NoError(err)
}

func (suite *TenableIntegrationSuite) TestGetLastScan() {
	image := &v1.Image{
		Name: &v1.ImageName{
			Sha:      "0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455",
			Registry: registry,
			Remote:   "srox/nginx",
			Tag:      "1.10",
		},
	}
	scan, err := suite.tenable.GetLastScan(image)
	suite.Nil(err)
	suite.NotNil(scan)
	if scan != nil {
		suite.NotEmpty(scan.Components)
	}
}
