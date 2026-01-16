package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/ioutils"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	supportedMediaTypes = set.NewFrozenStringSet(
		"text/spdx+json",        // Used by Sigstore/Cosign, not IANA registered.
		"application/spdx+json", // IANA registered type for SPDX JSON.
	)
)

type sbomScanHttpHandler struct {
	integrations integration.Set
}

var _ http.Handler = (*sbomScanHttpHandler)(nil)

func SBOMScanHandler(integrations integration.Set) http.Handler {
	return sbomScanHttpHandler{
		integrations: integrations,
	}
}

func (s sbomScanHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Verify Scanner V4 is enabled.
	if !features.ScannerV4.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("Scanner V4 is disabled."))
		return
	}

	if !features.SBOMScanning.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("SBOM Scanning is disabled."))
		return
	}

	// Only POST requests are supported.
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Validate the media type is supported.
	contentType := r.Header.Get("Content-Type")
	err := s.validateMediaType(contentType)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, fmt.Errorf("validating media type: %w", err))
		return
	}

	// Enforce maximum uncompressed request size to prevent excessive memory usage.
	// MaxBytesReader returns an error if the request body exceeds the limit.
	maxReqSizeBytes := env.SBOMScanMaxReqSizeBytes.IntegerSetting()
	limitedBody := http.MaxBytesReader(w, r.Body, int64(maxReqSizeBytes))

	// Add cancellation safety to prevent partial/corrupted data on interruption.
	// InterruptibleReader: Ensures clean termination without partial reads.
	body, interrupt := ioutils.NewInterruptibleReader(limitedBody)
	defer interrupt()

	// ContextBoundReader: Ensures reads fail fast when request context is canceled.
	// This prevents hanging reads on connection interruption
	readCtx, cancel := context.WithCancel(r.Context())
	defer cancel()
	body = ioutils.NewContextBoundReader(readCtx, body)

	sbomScanResponse, err := s.scanSBOM(readCtx, body, contentType)
	if err != nil {
		// Check if error is due to request body exceeding size limit.
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httputil.WriteGRPCStyleError(w, codes.InvalidArgument, fmt.Errorf("request body exceeds maximum size of %d bytes", maxBytesErr.Limit))
			return
		}
		httputil.WriteGRPCStyleError(w, codes.Internal, fmt.Errorf("scanning SBOM: %w", err))
		return
	}

	// Serialize the scan result to JSON using protojson for proper protobuf handling.
	// protojson handles protobuf-specific types (enums, oneof, etc.) correctly.
	jsonBytes, err := protojson.MarshalOptions{Multiline: true}.Marshal(sbomScanResponse)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, fmt.Errorf("serializing SBOM scan response: %w", err))
		return
	}

	// Set response headers and write JSON response.
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonBytes); err != nil {
		log.Warnw("writing SBOM scan response: %v", err)
		return
	}
}

// scanSBOM will request a scan of the SBOM from Scanner V4.
func (s sbomScanHttpHandler) scanSBOM(ctx context.Context, limitedReader io.Reader, contentType string) (*v1.SBOMScanResponse, error) {
	// Get reference to Scanner V4.
	scannerV4, dataSource, err := s.getScannerV4Integration()
	if err != nil {
		return nil, fmt.Errorf("getting Scanner V4 integration: %w", err)
	}

	// Scan the SBOM.
	sbomScanResponse, err := scannerV4.ScanSBOM(ctx, limitedReader, contentType)
	if err != nil {
		return nil, fmt.Errorf("scanning sbom: %w", err)
	}
	// Set the scan DataSource used to do the scan.
	if sbomScanResponse.GetScan() != nil {
		sbomScanResponse.GetScan().DataSource = dataSource
	}

	return sbomScanResponse, nil
}

// getScannerV4Integration returns the SBOM interface of Scanner V4.
func (s sbomScanHttpHandler) getScannerV4Integration() (scannerTypes.SBOMer, *storage.DataSource, error) {
	sbomer, dataSource, err := getScannerV4SBOMIntegration(s.integrations.ScannerSet())
	return sbomer, dataSource, err
}

// validateMediaType validates the media type from the content type header is supported.
func (s sbomScanHttpHandler) validateMediaType(contentType string) error {
	// Strip any parameters (e.g., charset) from the media type
	mediaType := strings.TrimSpace(strings.Split(contentType, ";")[0])
	if !supportedMediaTypes.Contains(mediaType) {
		return fmt.Errorf("unsupported media type %q, supported types %v", mediaType, supportedMediaTypes.AsSlice())
	}

	return nil
}
