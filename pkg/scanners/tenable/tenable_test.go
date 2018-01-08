package tenable

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

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
		fmt.Fprintf(w, scanPayload)
	})
	masterRouter.HandleFunc("/container-security/api/v1/container/list", func(w http.ResponseWriter, r *http.Request) {
		if err := handleAuth(r); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "{}")
	})

	masterServer := httptest.NewServer(masterRouter)

	// Set the global variable of the Tenable endpoint
	apiEndpoint = "http://" + masterServer.Listener.Addr().String()

	suite.server = masterServer

	protoScanner := &v1.Scanner{
		Config: map[string]string{
			"accessKey": "key1",
			"secretKey": "key2",
		},
	}

	var err error
	// newScanner is tested within setup
	suite.scanner, err = newScanner(protoScanner)
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
	image := &v1.Image{
		Registry: "",
		Remote:   "docker/nginx",
		Tag:      "1.10",
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
