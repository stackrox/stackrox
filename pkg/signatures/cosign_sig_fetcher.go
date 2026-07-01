package signatures

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	ociremote "github.com/sigstore/cosign/v3/pkg/oci/remote"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoutils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"golang.org/x/time/rate"
)

const fetchTimeout = 10 * time.Second

type cosignSignatureFetcher struct {
	// registryRateLimiter is a rate limiter for parallel calls to the registry. This will avoid reaching out to the
	// registry too many times leading to 429 errors.
	registryRateLimiter *rate.Limiter
}

var _ SignatureFetcher = (*cosignSignatureFetcher)(nil)

func newCosignSignatureFetcher() *cosignSignatureFetcher {
	return &cosignSignatureFetcher{
		registryRateLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
	}
}

var insecureDefaultTransport *http.Transport

func init() {
	insecureDefaultTransport = gcrRemote.DefaultTransport.(*http.Transport).Clone()
	insecureDefaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

// FetchSignatures implements the SignatureFetcher interface.
// Signatures are discovered via two methods concurrently: the legacy cosign tag-based
// path and the OCI 1.1 Referrers API. Results from both are merged and deduplicated.
// When one path fails but the other returns signatures, the error is logged and the
// successful results are returned. When both paths fail, the joined error is returned
// to the caller so retry logic can act on it. Unauthorized errors are wrapped as
// errox.NotAuthorized regardless of which path produced them.
// NOTE: No error will be returned when the image has no signature available.
func (c *cosignSignatureFetcher) FetchSignatures(ctx context.Context, image *storage.Image,
	fullImageName string, registry registryTypes.Registry, retryOpts ...retry.OptionsModifier,
) ([]*storage.Signature, error) {
	// Short-circuit for images that do not have V2 metadata associated with them. These would be older images manifest
	// schemes that are not supported by cosign, like the docker v1 manifest.
	if image.GetMetadata().GetV2() == nil {
		return nil, nil
	}

	// Since cosign makes heavy use of google/go-containerregistry, we need to parse the image's full name as a
	// name.Reference.
	imgRef, err := name.ParseReference(fullImageName)
	if err != nil {
		return nil, err
	}

	// Wait until the registry rate limiter allows entrance.
	err = c.registryRateLimiter.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for rate limiter entrance for registry %q: %w", registry.Name(), err)
	}

	// Inject the registry specific authentication options to the google/go-containerregistry client.
	remoteOpts := optionsFromRegistry(ctx, registry)
	log.Infof("FetchSignatures for %q: registry=%q hasAuth=%t",
		fullImageName, registry.Name(), len(remoteOpts) > 0)
	ociOpts := []ociremote.Option{ociremote.WithRemoteOptions(remoteOpts...)}

	// Fetch from both discovery methods concurrently. Each path retries independently
	// so a transient failure in one does not block the other.
	var (
		tagPayloads      []cosign.SignedPayload
		tagErr           error
		referrerPayloads []cosign.SignedPayload
		referrerErr      error
		wg               sync.WaitGroup
	)
	wg.Go(func() {
		tagPayloads, tagErr = fetchWithRetry(ctx, ociOpts, retryOpts, func(opts []ociremote.Option) ([]cosign.SignedPayload, error) {
			return fetchTagPayloads(image, imgRef, opts)
		})
	})
	wg.Go(func() {
		imgSHA := imgUtils.GetSHA(image)
		if imgSHA == "" {
			return
		}
		digestRef := imgRef.Context().Digest(imgSHA)
		referrerPayloads, referrerErr = fetchWithRetry(ctx, ociOpts, retryOpts, func(opts []ociremote.Option) ([]cosign.SignedPayload, error) {
			return fetchReferrerPayloads(ctx, digestRef, imgRef.Context(), opts)
		})
	})
	wg.Wait()

	// Merge payloads from both discovery methods.
	var allPayloads []cosign.SignedPayload
	if tagErr == nil {
		allPayloads = append(allPayloads, tagPayloads...)
		log.Infof("Tag-based discovery returned %d payload(s) for %q", len(tagPayloads), fullImageName)
	} else {
		log.Infof("Tag-based discovery failed for %q: %v", fullImageName, tagErr)
	}
	if referrerErr == nil {
		allPayloads = append(allPayloads, referrerPayloads...)
		log.Infof("Referrer-based discovery returned %d payload(s) for %q", len(referrerPayloads), fullImageName)
	} else {
		log.Infof("Referrer-based discovery failed for %q: %v", fullImageName, referrerErr)
	}
	log.Infof("Merged payload count for %q: %d", fullImageName, len(allPayloads))

	fetchErr := errors.Join(tagErr, referrerErr)
	if fetchErr != nil {
		if len(allPayloads) > 0 {
			log.Warnf("Partial signature discovery failure for %q: %v", fullImageName, fetchErr)
		} else if isUnauthorizedError(tagErr) || isUnauthorizedError(referrerErr) {
			return nil, errox.NotAuthorized.CausedBy(fetchErr)
		} else {
			return nil, fetchErr
		}
	}

	if len(allPayloads) == 0 {
		log.Infof("No signatures found for %q via either discovery method", fullImageName)
		return nil, nil
	}

	cosignSignatures := convertPayloadsToSignatures(allPayloads, fullImageName)
	if len(cosignSignatures) == 0 {
		return nil, nil
	}

	return protoutils.SliceUnique(cosignSignatures), nil
}

// fetchWithRetry calls fn, optionally wrapping it in retry logic with a per-attempt timeout.
// The timeout context is injected into the OCI options so fn does not need to handle contexts.
// When retryOpts is empty, fn is called directly without retry or timeout wrapping.
func fetchWithRetry(ctx context.Context, ociOpts []ociremote.Option, retryOpts []retry.OptionsModifier,
	fn func([]ociremote.Option) ([]cosign.SignedPayload, error),
) ([]cosign.SignedPayload, error) {
	if len(retryOpts) == 0 {
		return fn(ociOpts)
	}
	var payloads []cosign.SignedPayload
	err := retry.WithRetry(func() error {
		fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
		defer cancel()
		opts := append(slices.Clone(ociOpts),
			ociremote.WithMoreRemoteOptions(gcrRemote.WithContext(fetchCtx)))
		var fetchErr error
		payloads, fetchErr = fn(opts)
		return makeTransientErrorRetryable(fetchErr)
	}, retryOpts...)
	return payloads, err
}

// convertPayloadsToSignatures converts cosign signed payloads to storage signature protos.
func convertPayloadsToSignatures(payloads []cosign.SignedPayload, fullImageName string) []*storage.Signature {
	signatures := make([]*storage.Signature, 0, len(payloads))

	for _, signedPayload := range payloads {
		rawSig, err := base64.StdEncoding.DecodeString(signedPayload.Base64Signature)
		if err != nil {
			log.Errorf("Error during decoding of raw signature for image %q: %v",
				fullImageName, err)
			continue
		}

		certPEM, err := certificateFromSignedPayload(signedPayload)
		if err != nil {
			log.Errorf("Error during unmarshalling certificate to PEM for image %q: %v", fullImageName, err)
		}

		chainPEM, err := certificateChainFromSignedPayload(signedPayload)
		if err != nil {
			log.Errorf("Error during unmarshalling certificate chain to PEM for image %q: %v",
				fullImageName, err)
		}

		var rekorBundle []byte
		if signedPayload.Bundle != nil {
			rekorBundle, err = json.Marshal(signedPayload.Bundle)
			if err != nil {
				log.Errorf("Error during marshalling rekor bundle for image %q: %v", fullImageName, err)
			}
		}

		signatures = append(signatures, &storage.Signature{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig,
					SignaturePayload: signedPayload.Payload,
					CertPem:          certPEM,
					CertChainPem:     chainPEM,
					RekorBundle:      rekorBundle,
				},
			},
		})
	}

	return signatures
}

func certificateFromSignedPayload(sp cosign.SignedPayload) ([]byte, error) {
	if sp.Cert == nil {
		return nil, nil
	}

	pem, err := cryptoutils.MarshalCertificateToPEM(sp.Cert)
	if err != nil {
		return nil, err
	}
	return pem, nil
}

func certificateChainFromSignedPayload(sp cosign.SignedPayload) ([]byte, error) {
	if len(sp.Chain) == 0 {
		return nil, nil
	}

	pem, err := cryptoutils.MarshalCertificatesToPEM(sp.Chain)
	if err != nil {
		return nil, err
	}
	return pem, nil
}

// makeTransientErrorRetryable ensures that only transient errors are made retryable.
// Note: This takes into account the definition of the transport.Error, you can find more here:
// https://github.com/google/go-containerregistry/blob/f1fa40b162a1601a863364e8a2f63bbb9e4ff36e/pkg/v1/remote/transport/error.go#L90
func makeTransientErrorRetryable(err error) error {
	var transportErr *transport.Error
	var urlError *url.Error
	// We don't expect any transient errors that are coming from cosign at the moment.
	if errors.As(err, &transportErr) && transportErr.Temporary() {
		return retry.MakeRetryable(err)
	}

	if errors.As(err, &urlError) && urlError.Temporary() {
		return retry.MakeRetryable(err)
	}

	return err
}

func optionsFromRegistry(ctx context.Context, registry registryTypes.Registry) []gcrRemote.Option {
	registryCfg := registry.Config(ctx)
	if registryCfg == nil {
		return nil
	}

	var opts []gcrRemote.Option
	// Only create an option for the transport if we have custom authentication. By default, the transport will assume
	// anonymous access. We need to check both values since some registries use special strings for username, i.e.
	// GCR will use "_json_key" if used with a service account, quay.io registries will use "$oauthtoken".
	if registryCfg.Username != "" && registryCfg.Password != "" {
		// We are not changing the transport but keep default values, so it is safe to assign it to a variable without
		// potential race conditions.
		tr := gcrRemote.DefaultTransport
		if registryCfg.Insecure {
			tr = insecureDefaultTransport
		}
		// Instead of relying on gcrRemote's authentication, we are using the same authentication we already use for
		// our registries. The wrapped transport will ensure we are authenticated properly with all currently supported
		// registries. Ideally, we would in general use the same libraries for both, but cosign doesn't support
		// exchanging gcrRemote for now (we could however move to gcrRemote within the registry as well).
		opts = append(opts, gcrRemote.WithTransport(
			dockerRegistry.WrapTransport(tr, strings.TrimSuffix(registryCfg.URL, "/"),
				registryCfg.Username, registryCfg.Password)))
	}
	return opts
}

// isMissingSignatureError is checking whether the returned error indicates that the image has no signature available.
// If that is the case, we shouldn't treat it as an error, since images are allowed to have no signature associated with
// them.
func isMissingSignatureError(err error) bool {
	// Cosign doesn't provide error types we can use for checking, hence we need to do a string comparison.
	// Cosign ref:
	//  https://github.com/sigstore/cosign/blob/44f3814667ba6a398aef62814cabc82aee4896e5/pkg/cosign/fetch.go#L84-L86
	if strings.Contains(err.Error(), "no signatures associated") {
		return true
	}

	// Since we are using the transport created by the heroku client, it will be a mix of error types returned by
	// cosign. Cosign itself expects from registry operations the transport.Error, heroku-client will return a url.Error
	// instead. Because of this, cosign will potentially return the registry error instead of "no signatures associated"
	// error when no signatures are found. Hence, we have to check here the status code, if the code is
	// http.StatusNotFound we will indicate that no signatures are available.
	// Cosign ref:
	// https://github.com/sigstore/cosign/blob/b1024041754c8171375bf1a8411d86436c654b95/pkg/oci/remote/signatures.go#L35-L40
	return checkIfErrorContainsCode(err, http.StatusNotFound)
}

// isUnkownMimeTypeError is checking whether the error indicates that the image is an unknown mime type for cosign.
// Cosign itself only supports OCI or DockerV2 manifest schemes and will error out on any other, older manifest schemes.
// Cosign ref:
// https://github.com/sigstore/cosign/blob/6bfac1a470492d8964778b1b8c41e0056bf5dbdd/pkg/oci/remote/remote.go#L65-L76
func isUnknownMimeTypeError(err error) bool {
	// Cosign doesn't provide error types we can easily use for checking, hence we need to do a string comparison.
	// Cosign ref:
	// https://github.com/sigstore/cosign/blob/6bfac1a470492d8964778b1b8c41e0056bf5dbdd/pkg/oci/remote/remote.go#L76
	return strings.Contains(err.Error(), "unknown mime type")
}

// isUnauthorizedError is checking whether the returned error indicates that there was a http.StatusUnauthorized was
// returned during fetching of signatures.
func isUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}
	return checkIfErrorContainsCode(err, http.StatusUnauthorized, http.StatusForbidden)
}

// checkIfErrorContainsCode will try retrieve a http.StatusCode from the given error by casting the error to either
// transport.Error or registry.HttpStatusError and check whether the code is contained within a given list of codes.
// In case the error is matching one of these types and the code is contained within the given codes, true will be
// returned.
// If the error is not matching any of these types or the code is not contained in the given codes, false will be
// returned.
func checkIfErrorContainsCode(err error, codes ...int) bool {
	var transportErr *transport.Error
	var statusError *dockerRegistry.HttpStatusError

	// Transport error is returned by go-containerregistry for any errors occurred post authentication.
	if errors.As(err, &transportErr) {
		return slices.Index(codes, transportErr.StatusCode) != -1
	}

	// HttpStatusError is returned by heroku-client for any errors occurred during authentication.
	if errors.As(err, &statusError) && statusError.Response != nil {
		return slices.Index(codes, statusError.Response.StatusCode) != -1
	}

	return false
}
