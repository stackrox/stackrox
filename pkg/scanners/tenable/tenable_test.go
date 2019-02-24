package tenable

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/storage"
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

func TestTenableSuite(t *testing.T) {
	suite.Run(t, new(TenableSuite))
}

type TenableSuite struct {
	suite.Suite

	server  *httptest.Server
	scanner *tenable
}

func handleAuth(r *http.Request) error {
	if r.Header.Get("X-ApiKeys") != "accessKey=key1; secretKey=key2" {
		return fmt.Errorf("Not Authorized: %v", r.URL.String())
	}
	return nil
}

func (suite *TenableSuite) SetupSuite() {
	masterRouter := http.NewServeMux()
	// Handle
	masterRouter.HandleFunc("/container-security/api/v1/reports/by_image", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, scanPayload)
		suite.NoError(err)
	})
	masterRouter.HandleFunc("/container-security/api/v1/container/list", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, "{}")
		suite.NoError(err)
	})
	// Handle Registry ping
	masterRouter.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, "{}")
		suite.NoError(err)
	})
	// Handle
	masterRouter.HandleFunc("/v2/library/nginx/manifests/1.10", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, manifestPayload)
		suite.NoError(err)
	})

	masterServer := httptest.NewServer(masterRouter)

	// Set the global variable of the Tenable endpoint
	apiEndpoint = "http://" + masterServer.Listener.Addr().String()
	registryEndpoint = apiEndpoint

	suite.server = masterServer

	protoImageIntegration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Tenable{
			Tenable: &storage.TenableConfig{
				AccessKey: "key1",
				SecretKey: "key2",
			},
		},
	}

	var err error
	// newScanner is tested within setup
	suite.scanner, err = newScanner(protoImageIntegration)
	if err != nil {
		suite.FailNow("Could not setup DTR scanner: " + err.Error())
	}
}

func (suite *TenableSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *TenableSuite) TestTestFunction() {
	err := suite.scanner.Test()
	suite.NoError(err)
}

func (suite *TenableSuite) TestGetLastScan() {
	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "",
			Remote:   "docker/nginx",
			Tag:      "1.10",
		},
	}
	scan, err := suite.scanner.GetLastScan(image)
	suite.NoError(err)

	expectedTenableScan, err := getImageScan()
	suite.NoError(err)

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedImageScan := convertScanToImageScan(image, expectedTenableScan)

	// There is no ordering constraint on components as they are converted using a map so sort first and then compare
	sort.SliceStable(expectedImageScan.Components, func(i, j int) bool {
		return expectedImageScan.Components[i].Name < expectedImageScan.Components[j].Name
	})
	sort.SliceStable(scan.Components, func(i, j int) bool { return scan.Components[i].Name < scan.Components[j].Name })
	suite.Equal(expectedImageScan, scan)
}
