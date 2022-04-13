package quay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/suite"
)

const manifestPayload = `{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "digest": "sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455"
   },
   "layers": [
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "digest": "sha256:ef24d3d19d383c557b3bb92c21cc1b3e0c4ca6735160b6d3c684fb92ba0b3569"
      },
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 20197968,
         "digest": "sha256:96ebebd48bf5b659f6a6289aa67f5f6195f2aab6091df06beae8da160948e860"
      },
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 195,
         "digest": "sha256:783c67390305eb6df2f20ae03941343664fa3ca94b700eea449a46b1f6f686fe"
      }
   ]
}
`

func TestQuaySuite(t *testing.T) {
	suite.Run(t, new(QuaySuite))
}

type QuaySuite struct {
	suite.Suite

	server  *httptest.Server
	scanner *quay
}

func handleAuth(r *http.Request) error {
	if r.Header.Get("Authorization") != "Bearer token" {
		return fmt.Errorf("Not Authorized: %v", r.URL.String())
	}
	return nil
}

func (suite *QuaySuite) SetupSuite() {
	masterRouter := http.NewServeMux()
	// Handle
	// 	GET /api/v1/repository/{repository}/manifest/{manifestref}/security
	masterRouter.HandleFunc("/api/v1/repository/integration/nginx/manifest/sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455/security", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, scanPayload)
		suite.NoError(err)
	})
	// Handle Registry ping
	masterRouter.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, "{}")
		suite.NoError(err)
	})
	// Handle
	masterRouter.HandleFunc("/v2/integration/nginx/manifests/1.10", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, manifestPayload)
		suite.NoError(err)
	})

	masterServer := httptest.NewServer(masterRouter)

	// Set the global variable of the Quay endpoint
	suite.server = masterServer

	protoImageIntegration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Quay{
			Quay: &storage.QuayConfig{
				OauthToken: "token",
				Endpoint:   "http://" + masterServer.Listener.Addr().String(),
			},
		},
	}

	var err error
	// newScanner is tested within setup
	suite.scanner, err = newScanner(protoImageIntegration)
	if err != nil {
		suite.FailNow("Could not setup Quay scanner: " + err.Error())
	}
}

func (suite *QuaySuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *QuaySuite) TestScanTest() {
	err := suite.scanner.Test()
	suite.NoError(err)
}

func (suite *QuaySuite) TestGetScan() {
	image := &storage.Image{
		Id: "sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455",
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}
	scan, err := suite.scanner.GetScan(image)
	suite.NoError(err)

	expectedQuayScan, err := getImageScan()
	suite.NoError(err)

	// convert scans here. It relies on converting the scan but is not the conversion test.
	// skipping scan time check.
	expectedImageScan := convertScanToImageScan(image, expectedQuayScan)
	suite.Equal(expectedImageScan.Components, scan.Components)
	suite.Equal(expectedImageScan.OperatingSystem, scan.OperatingSystem)
	suite.Equal(expectedImageScan.DataSource, scan.DataSource)
}
