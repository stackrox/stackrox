package signatures

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
)

type cosignPublicKeySignatureFetcher struct{}

var _ SignatureFetcher = (*cosignPublicKeySignatureFetcher)(nil)

func newCosignPublicKeySignatureFetcher() *cosignPublicKeySignatureFetcher {
	return &cosignPublicKeySignatureFetcher{}
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
	registry registryTypes.ImageRegistry) ([]*storage.Signature, error) {
	// Since cosign makes heavy use of google/go-containerregistry, we need to parse the image's full name as a
	// name.Reference.
	imgFullName := image.GetName().GetFullName()
	imgRef, err := name.ParseReference(imgFullName)
	if err != nil {
		log.Errorf("Parsing image reference %q: %v", imgFullName, err)
		return nil, err
	}

	// Fetch the signatures by injecting the registry specific authentication options to the google/go-containerregistry
	// client.
	signedPayloads, err := cosign.FetchSignaturesForReference(ctx, imgRef,
		ociremote.WithRemoteOptions(optionsFromRegistry(registry)...))

	// Cosign will return an error in case no signature is associated, we don't want to return that error. Since no
	// error types are exposed need to check for string comparison.
	// Cosign ref:
	//  https://github.com/sigstore/cosign/blob/44f3814667ba6a398aef62814cabc82aee4896e5/pkg/cosign/fetch.go#L84-L86
	if err != nil && !strings.Contains(err.Error(), "no signatures associated") {
		log.Errorf("Fetching signature for image %q: %v", imgFullName, err)
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
	return err
}

func optionsFromRegistry(registry registryTypes.ImageRegistry) []gcrRemote.Option {
	registryCfg := registry.Config()
	if registryCfg == nil {
		return nil
	}

	var opts []gcrRemote.Option
	if registryCfg.Username != "" {
		opts = append(opts, gcrRemote.WithAuth(authn.FromConfig(authn.AuthConfig{
			Username: registryCfg.Username,
			Password: registryCfg.Password,
		})))
	}
	if registryCfg.Insecure {
		opts = append(opts, gcrRemote.WithTransport(insecureDefaultTransport))
	}

	return opts
}
