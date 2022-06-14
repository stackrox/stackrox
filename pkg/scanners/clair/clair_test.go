package clair

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestClairSuite(t *testing.T) {
	suite.Run(t, new(ClairSuite))
}

type ClairSuite struct {
	suite.Suite

	server  *httptest.Server
	scanner *clair
}

func (suite *ClairSuite) SetupSuite() {
	masterRouter := http.NewServeMux()
	// Handle getting layer endpoint
	masterRouter.HandleFunc("/v1/layers/sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455", func(w http.ResponseWriter, r *http.Request) {
		features, _ := mock.GetTestFeatures()
		bytes, _ := json.Marshal(&clairV1.LayerEnvelope{Layer: &clairV1.Layer{Features: features}})
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, string(bytes))
		suite.NoError(err)
	})
	// Handle namespace endpoint
	masterRouter.HandleFunc("/v1/namespaces", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	masterServer := httptest.NewServer(masterRouter)

	// Set the global variable of the Clair endpoint
	suite.server = masterServer

	protoScanner := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Clair{
			Clair: &storage.ClairConfig{
				Endpoint: "http://" + masterServer.Listener.Addr().String(),
			},
		},
	}

	var err error
	// newScanner is tested within setup
	suite.scanner, err = newScanner(protoScanner)
	if err != nil {
		suite.FailNow("Could not setup Clair scanner: " + err.Error())
	}
}

func (suite *ClairSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *ClairSuite) TestScanTest() {
	err := suite.scanner.Test()
	suite.NoError(err)
}

func (suite *ClairSuite) TestGetScan() {
	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
		Metadata: &storage.ImageMetadata{
			LayerShas: []string{
				"sha256:randomhashthatshouldnotbeused",
				"sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455",
			},
		},
	}
	scan, err := suite.scanner.GetScan(image)
	suite.NoError(err)

	features, _ := mock.GetTestFeatures()
	layerEnvelope := &clairV1.LayerEnvelope{Layer: &clairV1.Layer{Features: features}}

	// convert scans here. It relies on converting the scan but is not the conversion test
	expectedImageScan := convertLayerToImageScan(image, layerEnvelope)
	suite.Equal(expectedImageScan, scan)
}
