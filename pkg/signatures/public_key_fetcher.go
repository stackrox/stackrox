package signatures

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"golang.org/x/time/rate"
)

type cosignPublicKeySignatureFetcher struct {
	// registryRateLimiter is a rate limiter for parallel calls to the registry. This will avoid reaching out to the
	// registry too many times leading to 429 errors.
	registryRateLimiter *rate.Limiter
}

var _ SignatureFetcher = (*cosignPublicKeySignatureFetcher)(nil)

func newCosignPublicKeySignatureFetcher() *cosignPublicKeySignatureFetcher {
	return &cosignPublicKeySignatureFetcher{
		registryRateLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
	}
}

var (
	insecureDefaultTransport *http.Transport
)

func init() {
	insecureDefaultTransport = gcrRemote.DefaultTransport.Clone()
	insecureDefaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

// FetchSignatures implements the SignatureFetcher interface.
// The signature associated with the image will be fetched from the given registry.
// It will return the storage.ImageSignature and an error that indicated whether the fetching should be retried or not.
// NOTE: No error will be returned when the image has no signature available. All occurring errors will be logged.
func (c *cosignPublicKeySignatureFetcher) FetchSignatures(ctx context.Context, image *storage.Image,
	registry registryTypes.Registry) ([]*storage.Signature, error) {
	// Since cosign makes heavy use of google/go-containerregistry, we need to parse the image's full name as a
	// name.Reference.
	imgFullName := image.GetName().GetFullName()
	imgRef, err := name.ParseReference(imgFullName)
	if err != nil {
		return nil, err
	}

	// Wait until the registry rate limiter allows entrance.
	err = c.registryRateLimiter.Wait(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for rate limiter entrance for registry %q", registry.Name())
	}

	// Fetch the signatures by injecting the registry specific authentication options to the google/go-containerregistry
	// client.
	signedPayloads, err := cosign.FetchSignaturesForReference(ctx, imgRef,
		ociremote.WithRemoteOptions(optionsFromRegistry(registry)...))

	// Cosign will return an error in case no signature is associated, we don't want to return that error. Since no
	// error types are exposed need to check for string comparison.
	// Cosign ref:
	//  https://github.com/sigstore/cosign/blob/44f3814667ba6a398aef62814cabc82aee4896e5/pkg/cosign/fetch.go#L84-L86
	if err != nil && !isMissingSignatureError(err) {
		// Specifically mark an error as errox.NotAuthorized so we skip using the same credentials for fetching.
		// We can safely skip the potential marking of retryable errors as unauthorized errors are not transient.
		if isUnauthorizedError(err) {
			return nil, errox.NotAuthorized.CausedBy(err)
		}
		// Ensure we mark transient errors as retryable.
		return nil, makeTransientErrorRetryable(err)
	}

	// Short-circuit if no signatures are associated with the image.
	if len(signedPayloads) == 0 {
		return nil, nil
	}

	cosignSignatures := make([]*storage.Signature, 0, len(signedPayloads))

	for _, signedPayload := range signedPayloads {
		rawSig, err := base64.StdEncoding.DecodeString(signedPayload.Base64Signature)
		// We skip the invalid base64 signature and log its occurrence.
		if err != nil {
			log.Errorf("Error during decoding of raw signature for image %q: %v",
				imgFullName, err)
			continue
		}
		// Since we are only focusing on public keys, we are ignoring the certificate / rekor bundles associated with
		// the signature.
		cosignSignatures = append(cosignSignatures, &storage.Signature{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig,
					SignaturePayload: signedPayload.Payload,
				},
			},
		})
	}

	// Since we are skipping invalid base64 signatures, need to check the length of the result.
	if len(cosignSignatures) == 0 {
		return nil, nil
	}

	return cosignSignatures, nil
}

// makeTransientErrorRetryable ensures that only transient errors are made retryable.
// Note: This takes into account the definition of the transport.Error, you can find more here:
// https://github.com/google/go-containerregistry/blob/f1fa40b162a1601a863364e8a2f63bbb9e4ff36e/pkg/v1/remote/transport/error.go#L90
func makeTransientErrorRetryable(err error) error {
	// We don't expect any transient errors that are coming from cosign at the moment.
	if transportErr, ok := err.(*transport.Error); ok && transportErr.Temporary() {
		return retry.MakeRetryable(err)
	}

	// We need to check for url.Error as well, since heroku-client will return those.
	if urlErr := errToURLError(err); urlErr != nil && urlErr.Temporary() {
		return retry.MakeRetryable(err)
	}
	return err
}

func optionsFromRegistry(registry registryTypes.Registry) []gcrRemote.Option {
	registryCfg := registry.Config()
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
	if registryErr := errToRegistryError(err); registryErr != nil && registryErr.Response != nil &&
		registryErr.Response.StatusCode == http.StatusNotFound {
		return true
	}

	if transportErr, ok := err.(*transport.Error); ok && transportErr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

// isUnauthorizedError is checking whether the returned error indicates that there was a http.StatusUnauthorized was
// returned during fetching of signatures.
func isUnauthorizedError(err error) bool {
	if transportErr, ok := err.(*transport.Error); ok {
		return transportErr.StatusCode == http.StatusUnauthorized ||
			transportErr.StatusCode == http.StatusForbidden
	}

	if registryErr := errToRegistryError(err); registryErr != nil && registryErr.Response != nil {
		return registryErr.Response.StatusCode == http.StatusUnauthorized ||
			registryErr.Response.StatusCode == http.StatusForbidden
	}

	return false
}

// errToURLError is a helper for casting an error to a url.Error.
func errToURLError(err error) *url.Error {
	causeErr := errors.Cause(err)
	if urlErr, ok := causeErr.(*url.Error); ok {
		return urlErr
	}
	unwrapErr := errors.Unwrap(err)
	if urlErr, ok := unwrapErr.(*url.Error); ok {
		return urlErr
	}
	return nil
}

// errToRegistryError is a helper for casting an error to a registry.HttpStatusError.
func errToRegistryError(err error) *dockerRegistry.HttpStatusError {
	urlErr := errToURLError(err)
	if urlErr == nil || urlErr.Err == nil {
		return nil
	}

	if registryErr, ok := urlErr.Err.(*dockerRegistry.HttpStatusError); ok {
		return registryErr
	}
	return nil
}
