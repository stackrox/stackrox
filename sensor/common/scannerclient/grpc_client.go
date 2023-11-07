package scannerclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/scanner/pkg/client"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	log = logging.LoggerForModule()
)

// ScannerClient is the interface to remote image scanners used by Sensor.
type ScannerClient interface {
	GetImageAnalysis(context.Context, *storage.Image, *types.Config) (*ImageAnalysis, error)
	Close() error
}

// ImageAnalysis is the result of an image analysis.
type ImageAnalysis struct {
	ScanStatus scannerV1.ScanStatus
	ScanNotes  []scannerV1.Note
	// Fields for analysis results for each supported Scanner: Scanner V2 (v1 proto)
	// and Scanner V4 (v4 proto).
	V1Components *scannerV1.Components
	V4Contents   *v4.Contents
}

// v2Client is the client for StackRox Scanner based on Clair V2, also
// known as Scanner V2.
type v2Client struct {
	scannerV1.ImageScanServiceClient
	conn *grpc.ClientConn
}

// v4Client is the client for StackRox Scanner Indexer, also
// known as Scanner V4 Indexer.
type v4Client struct {
	// Sensor uses Scanner V4's client.
	client client.Scanner
}

// GetStatus returns the image analysis status
func (i *ImageAnalysis) GetStatus() scannerV1.ScanStatus {
	if i != nil {
		return i.ScanStatus
	}
	return scannerV1.ScanStatus_UNSET
}

// GetNotes returns the image analysis notes
func (i *ImageAnalysis) GetNotes() []scannerV1.Note {
	if i != nil {
		return i.ScanNotes
	}
	return nil
}

// GetComponents returns the image analysis V1 components, available if the
// underlying scan was done through legacy Scanner (aka. scanner-v2).
func (i *ImageAnalysis) GetComponents() *scannerV1.Components {
	if i != nil {
		return i.V1Components
	}
	return nil
}

// GetContents returns the image analysis V4 contents, available if the
// underlying scan was done through scanner-v4.
func (i *ImageAnalysis) GetContents() *v4.Contents {
	if i != nil {
		return i.V4Contents
	}
	return nil
}

// getScannerEndpoint reads and validate the Scanner gRPC endpoint setting. If
// the endpoint is empty or not configured properly (invalid) the value is
// returned and error will be set.
func getScannerEndpoint(s env.Setting) (string, error) {
	e := s.Setting()
	if e == "" {
		return e, errors.Errorf("%s is not set or empty", s.EnvVar())
	}
	e = strings.TrimPrefix(e, "https://")
	if strings.Contains(e, "://") {
		return e, errors.Errorf("%s has unsupported scheme: %s", s.EnvVar(), e)
	}
	return e, nil
}

// dial the scanner and returns a new ScannerClient.  The function is non-blocking and
// returns a non-nil error upon configuration error.
func dial(endpoint string, certID mtls.Subject) (*grpc.ClientConn, error) {
	tlsConfig, err := clientconn.TLSConfig(certID, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, fmt.Errorf("TLS config failed: %w", err)
	}
	// This is non-blocking. If we ever want this to block, then add the
	// grpc.WithBlock() DialOption.
	log.Infof("dialing scanner at %s", endpoint)
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, fmt.Errorf("grpc dial failed: %w", err)
	}
	return conn, nil
}

// dialV2 connect to scanner V1 gRPC and return a new ScannerClient.
func dialV2() (ScannerClient, error) {
	endpoint, err := getScannerEndpoint(env.ScannerSlimGRPCEndpoint)
	if err != nil {
		return nil, err
	}
	log.Infof("dialing scanner-v2 client: %s", endpoint)
	conn, err := dial(endpoint, mtls.ScannerSubject)
	if err != nil {
		return nil, err
	}
	return &v2Client{
		ImageScanServiceClient: scannerV1.NewImageScanServiceClient(conn),
		conn:                   conn,
	}, nil
}

// dialV4 connect to scanner V4 gRPC and return a new ScannerClient.
func dialV4() (ScannerClient, error) {
	ctx := context.Background()
	c, err := client.NewGRPCScanner(ctx,
		client.WithAddress(env.ScannerV4GRPCEndpoint.Setting()),
		// TODO: [ROX-19050] Set the Scanner V4 TLS validation when certificates are ready.
		client.SkipTLSVerification)
	if err != nil {
		return nil, err
	}
	return &v4Client{client: c}, nil
}

// GetImageAnalysis retrieves the image analysis results for the given image.
func (c *v2Client) GetImageAnalysis(ctx context.Context, image *storage.Image, cfg *types.Config) (*ImageAnalysis, error) {
	imgName := image.GetName().GetFullName()

	// The WaitForReady option will cause invocations to block (until server ready or
	// ctx done/expires) This was added so that on fresh installation of sensor when
	// scanner is not ready yet, local scans will not all fail and have to wait for
	// next reprocess to succeed
	resp, err := c.GetImageComponents(ctx, &scannerV1.GetImageComponentsRequest{
		Image: utils.GetFullyQualifiedFullName(image),
		Registry: &scannerV1.RegistryData{
			Url:      cfg.URL,
			Username: cfg.Username,
			Password: cfg.Password,
			Insecure: cfg.Insecure,
		},
	}, grpc.WaitForReady(true))
	if err != nil {
		log.Debugf("Unable to get image components from local Scanner for image %s: %v", imgName, err)
		return nil, errors.Wrap(err, "getting image components from scanner")
	}

	log.Debugf("Received image components from local Scanner for image %s", imgName)

	return &ImageAnalysis{
		ScanStatus:   resp.GetStatus(),
		ScanNotes:    resp.GetNotes(),
		V1Components: resp.GetComponents(),
	}, nil
}

// Close closes and cleanup the client connection.
func (c *v2Client) Close() error {
	return c.conn.Close()
}

func convertIndexReportToAnalysis(ir *v4.IndexReport) *ImageAnalysis {
	var st scannerV1.ScanStatus
	switch ir.GetState() {
	case "Terminal", "IndexError":
		st = scannerV1.ScanStatus_FAILED
	case "IndexFinished":
		st = scannerV1.ScanStatus_SUCCEEDED
	default:
		st = scannerV1.ScanStatus_ANALYZING
	}
	return &ImageAnalysis{
		ScanStatus: st,
		V4Contents: ir.GetContents(),
	}
}

func (c *v4Client) GetImageAnalysis(ctx context.Context, image *storage.Image, cfg *types.Config) (*ImageAnalysis, error) {
	var opts []name.Option
	if cfg.Insecure {
		opts = append(opts, name.Insecure)
	}
	n := fmt.Sprintf("%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), utils.GetSHA(image))
	ref, err := name.NewDigest(n, opts...)
	if err != nil {
		// TODO: ROX-19576: Is the assumption that images always have SHA correct?
		return nil, fmt.Errorf("creating digest reference: %w", err)
	}
	auth := authn.Basic{
		Username: cfg.Username,
		Password: cfg.Password,
	}
	ir, err := c.client.GetOrCreateImageIndex(ctx, ref, &auth)
	if err != nil {
		return nil, fmt.Errorf("get or create index report (reference: %q): %w", ref.Name(), err)
	}
	return convertIndexReportToAnalysis(ir), nil
}

// Close closes and cleanup the client connection.
func (c *v4Client) Close() error {
	return c.client.Close()
}
