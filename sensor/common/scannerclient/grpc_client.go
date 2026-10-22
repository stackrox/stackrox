package scannerclient

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	pkgscanner "github.com/stackrox/rox/pkg/scannerv4"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/sensor/common/centralcabundle"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
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
	ScanStatus     scannerV1.ScanStatus
	ScanNotes      []scannerV1.Note
	IndexerVersion string

	// Fields for analysis results for each supported Scanner: Scanner V2 (v1 proto)
	// and Scanner V4 (v4 proto).
	V1Components *scannerV1.Components
	V4Contents   *v4.Contents
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

// GetIndexerVersion returns the version of the indexer used to produce
// the associated index report.
func (i *ImageAnalysis) GetIndexerVersion() string {
	if i != nil {
		return i.IndexerVersion
	}

	return ""
}

// dialV4 connect to scanner V4 gRPC and return a new ScannerClient.
func dialV4() (ScannerClient, error) {
	ctx := context.Background()
	opts := []client.Option{client.WithIndexerAddress(env.ScannerV4IndexerEndpoint.Setting())}

	// Make sure the Scanner V4 client trusts all internal CAs.
	if cas := centralcabundle.Get(); len(cas) > 0 {
		log.Infof("Adding %d Central CA certificate(s) to Scanner V4 client", len(cas))
		opts = append(opts, client.WithRootCAs(cas...))
	}

	c, err := client.NewGRPCScanner(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "dialing scanner V4 gRPC client")
	}
	return &v4Client{client: c}, nil
}

func convertIndexReportToAnalysis(ir *v4.IndexReport, indexerVersion string) *ImageAnalysis {
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
		ScanStatus:     st,
		V4Contents:     ir.GetContents(),
		IndexerVersion: indexerVersion,
	}
}

func (c *v4Client) GetImageAnalysis(ctx context.Context, image *storage.Image, cfg *types.Config) (*ImageAnalysis, error) {
	ref, err := pkgscanner.DigestFromImage(image)
	if err != nil {
		return nil, errors.Wrap(err, "getting image digest for analysis")
	}

	var scannerVersion pkgscanner.Version
	auth := authn.Basic{
		Username: cfg.Username,
		Password: cfg.Password,
	}
	opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: cfg.GetInsecure()}
	ir, err := c.client.GetOrCreateImageIndex(ctx, ref, &auth, opt, client.Version(&scannerVersion))
	if err != nil {
		return nil, fmt.Errorf("get or create index report (reference: %q): %w", ref.Name(), err)
	}

	imageAnalysis := convertIndexReportToAnalysis(ir, scannerVersion.Indexer)
	log.Debugf("Converted index report from local Scanner V4 indexer to an image analysis: "+
		"image: %q, status: %q, indexerVersion: %q",
		image.GetName().GetFullName(),
		imageAnalysis.ScanStatus,
		imageAnalysis.IndexerVersion,
	)
	return imageAnalysis, nil
}

// Close closes and cleanup the client connection.
func (c *v4Client) Close() error {
	if err := c.client.Close(); err != nil {
		return errors.Wrap(err, "closing v4 scanner client")
	}
	return nil
}
