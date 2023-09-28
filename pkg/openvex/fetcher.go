package openvex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/hashicorp/go-multierror"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
	"github.com/openvex/go-vex/pkg/attestation"
	"github.com/openvex/go-vex/pkg/vex"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sliceutils"
	"golang.org/x/time/rate"
)

var (
	log = logging.LoggerForModule()

	_ Fetcher = (*openVexFetcher)(nil)
)

// Fetcher fetches OpenVEX reports associated with an image.
type Fetcher interface {
	Fetch(ctx context.Context, img *storage.Image, registry registryTypes.Registry) ([]*storage.OpenVex, error)
}

// NewFetcher creates a new fetcher for OpenVEX reports.
func NewFetcher() Fetcher {
	return &openVexFetcher{
		registryRateLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
	}
}

type openVexFetcher struct {
	registryRateLimiter *rate.Limiter
}

// Fetch an OpenVEX report.
func (o *openVexFetcher) Fetch(ctx context.Context, img *storage.Image, registry registryTypes.Registry) ([]*storage.OpenVex, error) {
	if img.GetMetadata().GetV2() == nil {
		return nil, nil
	}

	// TODO(dhaus): Not supporting multiple image names for simplicity.
	fullImageName := img.GetName().GetFullName()
	ref, err := name.ParseReference(fullImageName)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing image name %q", fullImageName)
	}

	if err := o.registryRateLimiter.Wait(ctx); err != nil {
		return nil, errors.Wrap(err, "waiting for rate limiter")
	}

	// Fetch the attestations for an image. Vex reports are attached as in-toto attestations to an image.
	// The usual jazz of auth options via ociremote.WithRemoteOptions is handled.
	payloads, err := cosign.FetchAttestationsForReference(ctx, ref, "",
		ociremote.WithRemoteOptions(optsFromRegistry(registry)...))
	if err != nil && checkErr(err) {
		return nil, errors.Wrapf(err, "fetching attestations for %q", fullImageName)
	}

	// Go through each in-toto attestation payload, if it's a VEX report unmarshall it and list them here.
	var vexReports []*vex.VEX
	var readVexErrors *multierror.Error
	for _, payload := range payloads {
		vexReport, err := readVexReport(payload)
		if err != nil {
			readVexErrors = multierror.Append(readVexErrors, err)
			continue
		}
		vexReports = append(vexReports, vexReport)
	}

	if err := readVexErrors.ErrorOrNil(); err != nil {
		log.Errorf("Some OpenVEX reports couldn't be read from the attestations: %v", err)
	}

	// Marshal the vex reports to raw bytes again for storage within the proto.
	var storageReports []*storage.OpenVex
	for _, vexReport := range vexReports {
		raw, err := json.Marshal(vexReport)
		if err != nil {
			log.Errorf("Unmarshalling OpenVEX report: %v", err)
		}
		storageReports = append(storageReports, &storage.OpenVex{OpenVexReport: raw})
	}

	return storageReports, nil
}

func readVexReport(payload cosign.AttestationPayload) (*vex.VEX, error) {
	// Skip if it's not an in-toto attestation (only those will contain VEX reports).
	if payload.PayloadType != "application/vnd.in-toto+json" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(payload.PayLoad)
	if err != nil {
		return nil, errors.Wrap(err, "decoding in-toto attestation")
	}

	log.Infof("Found in-toto attestation: %s", string(data))

	att := &attestation.Attestation{}
	if err := json.Unmarshal(data, att); err != nil {
		return nil, errors.Wrap(err, "unmarshalling in-toto attestation")
	}

	// If the in-toto attestation does not hold a VEX report, skip it.
	if att.PredicateType != vex.TypeURI {
		return nil, nil
	}

	return &att.Predicate, nil
}

func checkErr(err error) bool {
	// Good old cosign doesn't return a proper error when something does not exist :)
	if strings.Contains(err.Error(), "no attestations associated with") {
		return false
	}

	return !checkIfErrorContainsCode(err, http.StatusNotFound)
}

func checkIfErrorContainsCode(err error, codes ...int) bool {
	var transportErr *transport.Error
	var statusError *dockerRegistry.HttpStatusError

	// Transport error is returned by go-containerregistry for any errors occurred post authentication.
	if errors.As(err, &transportErr) {
		return sliceutils.Find(codes, transportErr.StatusCode) != -1
	}

	// HttpStatusError is returned by heroku-client for any errors occurred during authentication.
	if errors.As(err, &statusError) && statusError.Response != nil {
		return sliceutils.Find(codes, statusError.Response.StatusCode) != -1
	}

	return false
}

// Handles optionally adding authentication information compatible with our docker registry client.
func optsFromRegistry(registry registryTypes.Registry) []gcrRemote.Option {
	cfg := registry.Config()
	if cfg == nil {
		return nil
	}
	var opts []gcrRemote.Option
	if cfg.Username != "" && cfg.Password != "" {
		// TODO(dhaus): Not supporting insecure registries at the moment, probably shouldn't anyways.
		tr := gcrRemote.DefaultTransport

		opts = append(opts, gcrRemote.WithTransport(
			dockerRegistry.WrapTransport(tr, strings.TrimSuffix(cfg.Username, "/"),
				cfg.Username, cfg.Password)))
	}
	return opts
}
