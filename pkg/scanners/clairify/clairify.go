package clairify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v3"
	gogoProto "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	clairConv "github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/registries"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	clairV1 "github.com/stackrox/scanner/api/v1"
	clairGRPCV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/clairify/client"
	"github.com/stackrox/scanner/pkg/clairify/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	// defaultClientTimeout default timeout for scanner calls.
	defaultClientTimeout = 5 * time.Minute

	// nodeScanClientTimeout is used for node scanning operations, it is shorter than
	// the default because we expect it to have only the database on its path.
	nodeScanClientTimeout = 1 * time.Minute

	defaultMaxConcurrentScans = int64(30)
)

var (
	_ scannerTypes.Scanner                  = (*clairify)(nil)
	_ scannerTypes.ImageVulnerabilityGetter = (*clairify)(nil)

	log             = logging.LoggerForModule()
	scannerEndpoint = fmt.Sprintf("scanner.%s.svc", env.Namespace.Setting())
)

// GetScannerEndpoint returns the scanner endpoint with a configured namespace. env.ScannerGRPCEndpoint is only used by Sensor.
func GetScannerEndpoint() string {
	return scannerEndpoint
}

// Creator provides the type scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (scannerTypes.Scanner, error)) {
	return scannerTypes.Clairify, func(integration *storage.ImageIntegration) (scannerTypes.Scanner, error) {
		return newScanner(integration, set)
	}
}

// NodeScannerCreator provides the type scanners.NodeScannerCreator to add to the scanners registry.
func NodeScannerCreator() (string, func(integration *storage.NodeIntegration) (scannerTypes.NodeScanner, error)) {
	return scannerTypes.Clairify, func(integration *storage.NodeIntegration) (scannerTypes.NodeScanner, error) {
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

	pingServiceClient      clairGRPCV1.PingServiceClient
	imageScanServiceClient clairGRPCV1.ImageScanServiceClient
	nodeScanServiceClient  clairGRPCV1.NodeScanServiceClient
	protoNodeIntegration   *storage.NodeIntegration

	orchestratorScanServiceClient clairGRPCV1.OrchestratorScanServiceClient
	protoOrchestratorIntegration  *storage.OrchestratorIntegration
}

func newScanner(protoImageIntegration *storage.ImageIntegration, activeRegistries registries.Set) (*clairify, error) {
	conf := protoImageIntegration.GetClairify()
	if conf == nil {
		return nil, errors.New("Clairify configuration required")
	}
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
		Timeout: defaultClientTimeout,
		Transport: &http.Transport{
			DialContext:     dialer.DialContext,
			TLSClientConfig: tlsConfig,
			Proxy:           proxy.FromConfig(),
		},
	}

	gRPCConnection, err := createGRPCConnectionToScanner(conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make gRPC connection to Scanner")
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

		imageScanServiceClient: clairGRPCV1.NewImageScanServiceClient(gRPCConnection),

		ScanSemaphore: scannerTypes.NewSemaphoreWithValue(numConcurrentScans),
	}
	return scanner, nil
}

func createGRPCConnectionToScanner(conf *storage.ClairifyConfig) (*grpc.ClientConn, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	tlsConfig, err := getTLSConfig()
	if err != nil {
		return nil, err
	}

	// Checking for an empty endpoint can't be removed because of backward-compatibility. Existing image
	// integrations are configured in the database on Central's startup and are not updated dynamically.
	endpoint := conf.GetGrpcEndpoint()
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s:8443", GetScannerEndpoint())
	}

	// Note: it is possible we call `grpc.Dial` multiple times per endpoint,
	// but this is rather minimal, so it's ok.
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

func newNodeScanner(protoNodeIntegration *storage.NodeIntegration) (*clairify, error) {
	conf := protoNodeIntegration.GetClairify()
	if conf == nil {
		return nil, errors.New("scanner configuration required")
	}
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	gRPCConnection, err := createGRPCConnectionToScanner(conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make gRPC connection to Scanner")
	}

	pingServiceClient := clairGRPCV1.NewPingServiceClient(gRPCConnection)
	scanServiceClient := clairGRPCV1.NewNodeScanServiceClient(gRPCConnection)
	// required as RHCOS scanning uses ImageScan API
	imageScanServiceClient := clairGRPCV1.NewImageScanServiceClient(gRPCConnection)

	return &clairify{
		NodeScanSemaphore:      scannerTypes.NewNodeSemaphoreWithValue(defaultMaxConcurrentScans),
		conf:                   conf,
		pingServiceClient:      pingServiceClient,
		nodeScanServiceClient:  scanServiceClient,
		imageScanServiceClient: imageScanServiceClient,
		protoNodeIntegration:   protoNodeIntegration,
	}, nil
}

func getTLSConfig() (*tls.Config, error) {
	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientTimeout)
	defer cancel()
	_, err := c.pingServiceClient.Ping(ctx, &clairGRPCV1.Empty{})
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

	noteSet := set.NewStringSet()
	var notes []storage.ImageScan_Note
	for _, note := range layerEnvelope.Notes {
		n := convertNote(note)
		if n == -1 {
			continue
		}

		if noteSet.Add(n.String()) {
			notes = append(notes, n)
		}
	}

	if isPartialScan(noteSet) {
		notes = append(notes, storage.ImageScan_PARTIAL_SCAN_DATA)
	}

	os := stringutils.OrDefault(layerEnvelope.Layer.NamespaceName, "unknown")
	return &storage.ImageScan{
		OperatingSystem: os,
		ScanTime:        gogoProto.TimestampNow(),
		ScannerVersion:  layerEnvelope.ScannerVersion,
		Components:      clairConv.ConvertFeatures(image, layerEnvelope.Layer.Features, os),
		Notes:           notes,
	}
}

func isPartialScan(notes set.StringSet) bool {
	osCVEsUnavailable := notes.Contains(storage.ImageScan_OS_CVES_UNAVAILABLE.String())
	languageCVEsUnavailable := notes.Contains(storage.ImageScan_LANGUAGE_CVES_UNAVAILABLE.String())
	certifiedRHELUnavailable := notes.Contains(storage.ImageScan_CERTIFIED_RHEL_SCAN_UNAVAILABLE.String())

	// != simulates XOR for bool values.
	// When both osCVEsUnavailable and languageCVEsUnavailable are true, we have no scan results.
	// When they are both false, we have full scan results.
	// Otherwise, we have partial results.
	if osCVEsUnavailable != languageCVEsUnavailable {
		return true
	}

	// We are able to perform a full scan, but the results are not certified by Red Hat.
	if !osCVEsUnavailable && !languageCVEsUnavailable && certifiedRHELUnavailable {
		return true
	}

	return false
}

func convertNote(note clairV1.Note) storage.ImageScan_Note {
	switch note {
	case clairV1.OSCVEsUnavailable:
		return storage.ImageScan_OS_CVES_UNAVAILABLE
	case clairV1.OSCVEsStale:
		return storage.ImageScan_OS_CVES_STALE
	case clairV1.LanguageCVEsUnavailable:
		return storage.ImageScan_LANGUAGE_CVES_UNAVAILABLE
	case clairV1.CertifiedRHELScanUnavailable:
		return storage.ImageScan_CERTIFIED_RHEL_SCAN_UNAVAILABLE
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
func (c *clairify) getScan(image *storage.Image, opts *types.GetImageDataOpts) (*clairV1.LayerEnvelope, error) {
	if layerEnv, err := c.httpClient.RetrieveImageDataBySHA(utils.GetSHA(image), opts); err == nil {
		return layerEnv, nil
	}
	return c.httpClient.RetrieveImageDataByName(v1ImageToClairifyImage(image), opts)
}

func (c *clairify) getInitialScanResults(img *storage.Image) (*clairV1.LayerEnvelope, error) {
	sha := utils.GetSHA(img)
	var opts types.GetImageDataOpts
	layerEnv, err := c.httpClient.RetrieveImageDataBySHA(sha, &opts)
	if err != nil {
		return nil, err
	}
	for _, note := range layerEnv.Notes {
		if note == clairV1.CertifiedRHELScanUnavailable {
			// We were unable to get certified scan results.
			// Try getting uncertified results instead.
			// This will also return a clairV1.CertifiedRHELScanUnavailable note.
			log.Debugf("Image %v is out of Red Hat Scanner Certification scope. Retrying fetch for uncertified results", v1ImageToClairifyImage(img))
			opts.UncertifiedRHELResults = true
			layerEnv, err = c.httpClient.RetrieveImageDataBySHA(sha, &opts)
		}
	}

	return layerEnv, err
}

// GetScan retrieves the most recent scan
func (c *clairify) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	// If we haven't retrieved any metadata then we won't be able to scan with Clairify
	if image.GetMetadata() == nil {
		return nil, nil
	}
	// If not found by digest, then should trigger a scan
	layerEnv, err := c.getInitialScanResults(image)
	if err != nil {
		if err != client.ErrorScanNotFound {
			return nil, err
		}
		var opts types.GetImageDataOpts
		layerEnv, err = c.scanImage(image, opts)
		if err != nil {
			return nil, err
		}
		for _, note := range layerEnv.Notes {
			if note == clairV1.CertifiedRHELScanUnavailable {
				// We were unable to get certified scan results.
				// Try getting uncertified results instead.
				// This will also return a clairV1.CertifiedRHELScanUnavailable note.
				opts.UncertifiedRHELResults = true
				log.Debugf("Image %v is out of Red Hat Scanner Certification scope. Retrying scan for uncertified results", v1ImageToClairifyImage(image))
				layerEnv, err = c.scanImage(image, opts)
				if err != nil {
					return nil, err
				}

				break
			}
		}

	}
	scan := convertLayerToImageScan(image, layerEnv)
	if scan == nil {
		return nil, errors.New("malformed response from scanner")
	}
	return scan, nil
}

func (c *clairify) scanImage(image *storage.Image, opts types.GetImageDataOpts) (*clairV1.LayerEnvelope, error) {
	if err := c.addScan(image, opts.UncertifiedRHELResults); err != nil {
		return nil, err
	}
	layerEnv, err := c.getScan(image, &opts)
	if err != nil {
		return nil, err
	}

	return layerEnv, nil
}

func (c *clairify) addScan(image *storage.Image, uncertifiedRHEL bool) error {
	rc := c.activeRegistries.GetRegistryMetadataByImage(image)
	if rc == nil {
		return nil
	}

	_, err := c.httpClient.AddImage(rc.Username, rc.Password, &types.ImageRequest{
		Image:               utils.GetFullyQualifiedFullName(image),
		Registry:            rc.URL,
		Insecure:            rc.Insecure,
		UncertifiedRHELScan: uncertifiedRHEL,
	})
	return err
}

// GetVulnerabilities retrieves the vulnerabilities present in the given image
// represented by the given components and scan notes.
func (c *clairify) GetVulnerabilities(image *storage.Image, components *clairGRPCV1.Components, notes []clairGRPCV1.Note) (*storage.ImageScan, error) {
	req := &clairGRPCV1.GetImageVulnerabilitiesRequest{
		Components: components,
		Notes:      notes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientTimeout)
	defer cancel()
	resp, err := c.imageScanServiceClient.GetImageVulnerabilities(ctx, req)
	if err != nil {
		return nil, err
	}

	return convertImageToImageScan(image.GetMetadata(), resp.GetImage()), nil
}

func retryOnGRPCErrors(ctx context.Context, name string, f func() error) error {
	op := func() error {
		err := f()
		if err != nil {
			e, _ := status.FromError(err)
			switch e.Code() {
			// Unavailable indicates the service is currently unavailable. The error code is
			// set by the gRPC framework during failed connections, abrupt shutdown of a
			// server process or network connection.
			//
			// Aborted is not generated by the gRPC framework, but may be used by the server
			// to retry on certain conditions that are retriable, e.g concurrency issue,
			// sequencer check failures, transaction aborts, etc.
			case codes.Aborted, codes.Unavailable:
				return err
			default:
				return backoff.Permanent(err)
			}
		}
		return err
	}
	notify := func(err error, duration time.Duration) {
		log.Warnf("calling %s() (retrying in %s): %v", name, duration, err)
	}
	eb := backoff.NewExponentialBackOff()
	return backoff.RetryNotify(op, backoff.WithContext(eb, ctx), notify)
}

func (c *clairify) GetNodeInventoryScan(node *storage.Node, inv *storage.NodeInventory) (*storage.NodeScan, error) {
	req := convertNodeToVulnRequest(node, inv)
	ctx, cancel := context.WithTimeout(context.Background(), nodeScanClientTimeout)
	defer cancel()
	log.Debugf("Calling GetNodeVulnerabilities with node inventory: %v", req.GetComponents())
	var resp *clairGRPCV1.GetNodeVulnerabilitiesResponse
	err := retryOnGRPCErrors(ctx, "GetNodeVulnerabilities", func() (err error) {
		resp, err = c.nodeScanServiceClient.GetNodeVulnerabilities(ctx, req)
		return err
	})
	if err != nil {
		return nil, err
	}
	log.Debugf("Got reply GetNodeVulnerabilities with features: %v", resp.GetFeatures())

	scan := convertVulnResponseToNodeScan(req, resp)
	if scan == nil {
		return nil, errors.New("malformed vuln response from scanner")
	}

	return scan, nil
}

// GetNodeScan retrieves the most recent node scan
func (c *clairify) GetNodeScan(node *storage.Node) (*storage.NodeScan, error) {
	return c.GetNodeInventoryScan(node, nil)
}

// Match decides if the image is contained within this scanner
func (c *clairify) Match(image *storage.ImageName) bool {
	return c.activeRegistries.Match(image)
}

// Type returns the stringified type of this scanner
func (c *clairify) Type() string {
	return scannerTypes.Clairify
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

// OrchestratorScannerCreator provides creator for OrchestratorScanner
func OrchestratorScannerCreator() (string, func(integration *storage.OrchestratorIntegration) (scannerTypes.OrchestratorScanner, error)) {
	return scannerTypes.Clairify, func(integration *storage.OrchestratorIntegration) (scannerTypes.OrchestratorScanner, error) {
		return newOrchestratorScanner(integration)
	}
}

func newOrchestratorScanner(integration *storage.OrchestratorIntegration) (*clairify, error) {
	conf := integration.GetClairify()
	if conf == nil {
		return nil, errors.New("scanner configuration required")
	}

	gRPCConnection, err := createGRPCConnectionToScanner(conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make gRPC connection to Scanner")
	}

	return &clairify{
		ScanSemaphore:                 scannerTypes.NewSemaphoreWithValue(defaultMaxConcurrentScans),
		conf:                          conf,
		protoOrchestratorIntegration:  integration,
		orchestratorScanServiceClient: clairGRPCV1.NewOrchestratorScanServiceClient(gRPCConnection),
	}, nil
}

// KubernetesScan retrieves the most recent orchestrator scan from scanner
func (c *clairify) KubernetesScan(version string) (map[string][]*storage.EmbeddedVulnerability, error) {
	req := &clairGRPCV1.GetKubeVulnerabilitiesRequest{
		KubernetesVersion: version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientTimeout)
	defer cancel()

	resp, err := c.orchestratorScanServiceClient.GetKubeVulnerabilities(ctx, req)
	if err != nil {
		return nil, err
	}

	results := map[string][]*storage.EmbeddedVulnerability{
		kubernetes.KubeAPIServer:         convertK8sVulns(resp.ApiserverVulnerabilities),
		kubernetes.KubeAggregator:        convertK8sVulns(resp.AggregatorVulnerabilities),
		kubernetes.KubeControllerManager: convertK8sVulns(resp.ControllerManagerVulnerabilities),
		kubernetes.KubeScheduler:         convertK8sVulns(resp.SchedulerVulnerabilities),
		kubernetes.Generic:               convertK8sVulns(resp.GenericVulnerabilities),
	}

	return results, nil
}

// IstioScan retrieves the most recent Istio scan from scanner
func (c *clairify) IstioScan(version string) ([]*storage.EmbeddedVulnerability, error) {

	req := &clairGRPCV1.GetIstioVulnerabilitiesRequest{
		IstioVersion: version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientTimeout)
	defer cancel()
	resp, err := c.orchestratorScanServiceClient.GetIstioVulnerabilities(ctx, req)
	if err != nil {
		return nil, err
	}

	res := convertIstioVulns(resp.GetVulnerabilities())

	return res, nil
}

// OpenShiftScan retrieves OpenShift scan from scanner
func (c *clairify) OpenShiftScan(version string) ([]*storage.EmbeddedVulnerability, error) {
	req := &clairGRPCV1.GetOpenShiftVulnerabilitiesRequest{
		OpenShiftVersion: version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientTimeout)
	defer cancel()

	resp, err := c.orchestratorScanServiceClient.GetOpenShiftVulnerabilities(ctx, req)
	if err != nil {
		return nil, err
	}

	results := convertVulnerabilities(resp.Vulnerabilities, storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY)

	return results, nil
}
