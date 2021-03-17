package clairify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	gogoProto "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	clairConv "github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/registries"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	clairV1 "github.com/stackrox/scanner/api/v1"
	clairGRPCV1 "github.com/stackrox/scanner/generated/shared/api/v1"
	"github.com/stackrox/scanner/pkg/clairify/client"
	"github.com/stackrox/scanner/pkg/clairify/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	typeString = "clairify"

	clientTimeout             = 5 * time.Minute
	defaultMaxConcurrentScans = int64(30)
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (scannerTypes.Scanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (scannerTypes.Scanner, error) {
		return newScanner(integration, set)
	}
}

// NodeScannerCreator provides the type scanners.NodeScannerCreator to add to the scanners registry.
func NodeScannerCreator() (string, func(integration *storage.NodeIntegration) (scannerTypes.NodeScanner, error)) {
	return typeString, func(integration *storage.NodeIntegration) (scannerTypes.NodeScanner, error) {
		return newNodeScanner(integration)
	}
}

type clairify struct {
	scannerTypes.ScanSemaphore
	scannerTypes.NodeScanSemaphore

	conf *storage.ClairifyConfig

	httpClient            *client.Clairify
	protoImageIntegration *storage.ImageIntegration
	activeRegistries      registries.Set

	pingServiceClient    clairGRPCV1.PingServiceClient
	scanServiceClient    clairGRPCV1.NodeScanServiceClient
	protoNodeIntegration *storage.NodeIntegration
}

func newScanner(protoImageIntegration *storage.ImageIntegration, activeRegistries registries.Set) (*clairify, error) {
	clairifyConfig, ok := protoImageIntegration.IntegrationConfig.(*storage.ImageIntegration_Clairify)
	if !ok {
		return nil, errors.New("Clairify configuration required")
	}
	conf := clairifyConfig.Clairify
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	endpoint := urlfmt.FormatURL(conf.Endpoint, urlfmt.InsecureHTTP, urlfmt.NoTrailingSlash)

	dialer := net.Dialer{
		Timeout: 2 * time.Second,
	}

	tlsConfig, err := getTLSConfig()
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: clientTimeout,
		Transport: &http.Transport{
			DialContext:     dialer.DialContext,
			TLSClientConfig: tlsConfig,
			Proxy:           proxy.FromConfig(),
		},
	}

	numConcurrentScans := defaultMaxConcurrentScans
	if conf.GetNumConcurrentScans() != 0 {
		numConcurrentScans = int64(conf.GetNumConcurrentScans())
	}

	scanner := &clairify{
		httpClient:            client.NewWithClient(endpoint, httpClient),
		conf:                  conf,
		protoImageIntegration: protoImageIntegration,
		activeRegistries:      activeRegistries,

		ScanSemaphore: scannerTypes.NewSemaphoreWithValue(numConcurrentScans),
	}
	return scanner, nil
}

func newNodeScanner(protoNodeIntegration *storage.NodeIntegration) (*clairify, error) {
	if !features.HostScanning.Enabled() {
		return nil, errors.New("node scanning is not currently enabled")
	}

	conf := protoNodeIntegration.GetClairify()
	if conf == nil {
		return nil, errors.New("scanner configuration required")
	}
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	tlsConfig, err := getTLSConfig()
	if err != nil {
		return nil, err
	}

	endpoint := conf.GetGrpcEndpoint()
	if endpoint == "" {
		endpoint = fmt.Sprintf("scanner.%s:8443", env.Namespace.Setting())
	}

	gRPCConnection, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to make gRPC connection to Scanner")
	}

	pingServiceClient := clairGRPCV1.NewPingServiceClient(gRPCConnection)
	scanServiceClient := clairGRPCV1.NewNodeScanServiceClient(gRPCConnection)

	return &clairify{
		NodeScanSemaphore:    scannerTypes.NewNodeSemaphoreWithValue(defaultMaxConcurrentScans),
		conf:                 conf,
		pingServiceClient:    pingServiceClient,
		scanServiceClient:    scanServiceClient,
		protoNodeIntegration: protoNodeIntegration,
	}, nil
}

func getTLSConfig() (*tls.Config, error) {
	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize TLS config")
	}
	return tlsConfig, nil
}

// Test initiates a test of the Clairify Scanner which verifies that we have the proper scan permissions
func (c *clairify) Test() error {
	return c.httpClient.Ping()
}

// TestNodeScanner initiates a test of the Clairify Scanner which verifies
// that we have the proper scan permissions for node scanning
func (c *clairify) TestNodeScanner() error {
	_, err := c.pingServiceClient.Ping(context.Background(), &clairGRPCV1.Empty{})
	return err
}

func validateConfig(c *storage.ClairifyConfig) error {
	if c.GetEndpoint() == "" {
		return errors.New("endpoint parameter must be defined for Clairify")
	}
	return nil
}

func convertLayerToImageScan(image *storage.Image, layerEnvelope *clairV1.LayerEnvelope) *storage.ImageScan {
	if layerEnvelope == nil || layerEnvelope.Layer == nil {
		return nil
	}

	var notes []storage.ImageScan_Note
	var hasStaleCVEs bool
	for _, note := range layerEnvelope.Notes {
		n := convertNote(note)
		if n == -1 {
			continue
		}
		if n == storage.ImageScan_OS_CVES_STALE {
			hasStaleCVEs = true
		}
		notes = append(notes, n)
	}

	if len(layerEnvelope.Notes) == 1 && !hasStaleCVEs {
		notes = append(notes, storage.ImageScan_PARTIAL_SCAN_DATA)
	}

	return &storage.ImageScan{
		OperatingSystem: stringutils.OrDefault(layerEnvelope.Layer.NamespaceName, "unknown"),
		ScanTime:        gogoProto.TimestampNow(),
		Components:      clairConv.ConvertFeatures(image, layerEnvelope.Layer.Features),
		Notes:           notes,
	}
}

func convertNote(note clairV1.Note) storage.ImageScan_Note {
	switch note {
	case clairV1.OSCVEsUnavailable:
		return storage.ImageScan_OS_CVES_UNAVAILABLE
	case clairV1.OSCVEsStale:
		return storage.ImageScan_OS_CVES_STALE
	case clairV1.LanguageCVEsUnavailable:
		return storage.ImageScan_LANGUAGE_CVES_UNAVAILABLE
	default:
		return -1
	}
}

func v1ImageToClairifyImage(i *storage.Image) *types.Image {
	return &types.Image{
		SHA:      i.GetId(),
		Registry: i.GetName().GetRegistry(),
		Remote:   i.GetName().GetRemote(),
		Tag:      i.GetName().GetTag(),
	}
}

// Try many ways to retrieve a sha
func (c *clairify) getScan(image *storage.Image) (*clairV1.LayerEnvelope, error) {
	if env, err := c.httpClient.RetrieveImageDataBySHA(utils.GetSHA(image), true, true); err == nil {
		return env, nil
	}
	return c.httpClient.RetrieveImageDataByName(v1ImageToClairifyImage(image), true, true)
}

// GetScan retrieves the most recent scan
func (c *clairify) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	// If we haven't retrieved any metadata then we won't be able to scan with Clairify
	if image.GetMetadata() == nil {
		return nil, nil
	}
	// If not found by digest, then should trigger a scan
	env, err := c.httpClient.RetrieveImageDataBySHA(utils.GetSHA(image), true, true)
	if err != nil {
		if err != client.ErrorScanNotFound {
			return nil, err
		}
		if err := c.scan(image); err != nil {
			return nil, err
		}
		env, err = c.getScan(image)
		if err != nil {
			return nil, err
		}
	}
	scan := convertLayerToImageScan(image, env)
	if scan == nil {
		return nil, errors.New("malformed response from scanner")
	}
	return scan, nil
}

func (c *clairify) scan(image *storage.Image) error {
	rc := c.activeRegistries.GetRegistryMetadataByImage(image)
	if rc == nil {
		return nil
	}

	_, err := c.httpClient.AddImage(rc.Username, rc.Password, &types.ImageRequest{
		Image:    utils.GetFullyQualifiedFullName(image),
		Registry: rc.URL,
		Insecure: rc.Insecure})
	return err
}

// GetNodeScan retrieves the most recent node scan
func (c *clairify) GetNodeScan(node *storage.Node) (*storage.NodeScan, error) {
	if !features.HostScanning.Enabled() {
		return nil, errors.New("Host scanning is disabled")
	}

	req := convertNodeToVulnRequest(node)
	resp, err := c.scanServiceClient.GetNodeVulnerabilities(context.Background(), req)
	if err != nil {
		return nil, err
	}

	scan := convertVulnResponseToNodeScan(req, resp)
	if scan == nil {
		return nil, errors.New("malformed vuln response from scanner")
	}

	return scan, nil
}

// Match decides if the image is contained within this scanner
func (c *clairify) Match(image *storage.ImageName) bool {
	return c.activeRegistries.Match(image)
}

// Type returns the stringified type of this scanner
func (c *clairify) Type() string {
	return typeString
}

// Name returns the integration's name
func (c *clairify) Name() string {
	return c.protoImageIntegration.GetName()
}

// GetVulnDefinitionsInfo gets the vulnerability definition metadata.
func (c *clairify) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	info, err := c.httpClient.GetVulnDefsMetadata()
	if err != nil {
		return nil, err
	}

	return &v1.VulnDefinitionsInfo{
		LastUpdatedTimestamp: info.GetLastUpdatedTime(),
	}, nil
}
