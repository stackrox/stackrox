package signatures

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
)

type cosignPublicKeyFetcher struct{}

var _ SignatureFetcher = (*cosignPublicKeyFetcher)(nil)

func newCosignPublicKeyFetcher() *cosignPublicKeyFetcher {
	return &cosignPublicKeyFetcher{}
}

// FetchSignature implements the SignatureFetcher interface.
// The signature associated with the image will be fetched from the given registry.
// It will return the storage.ImageSignature and a boolean, indicating whether any signatures were found or not.
func (c *cosignPublicKeyFetcher) FetchSignature(ctx context.Context, image *storage.Image,
	registry registryTypes.ImageRegistry) (*storage.ImageSignature, bool) {
	// Since cosign makes heavy use of google/go-containerregistry, we need to parse the image's full name as a
	// name.Reference.
	imgFullName := image.GetName().GetFullName()
	imgRef, err := name.ParseReference(imgFullName)
	if err != nil {
		log.Errorf("Parsing image reference %q: %v", imgFullName, err)
		return nil, false
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
		return nil, false
	}

	// Short-circuit if no signatures are associated with the image.
	if len(signedPayloads) == 0 {
		return nil, false
	}

	cosignSignatures := make([]*storage.Signature, 0, len(signedPayloads))

	for _, signedPayload := range signedPayloads {
		rawSig, err := base64.StdEncoding.DecodeString(signedPayload.Base64Signature)
		// We skip the invalid base64 signature and log its occurrence.
		if err != nil {
			log.Errorf("Error during decoding of raw signature for image %q: %v",
				imgFullName, err)
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
		return nil, false
	}

	return &storage.ImageSignature{
		Signatures: cosignSignatures,
	}, true
}

func optionsFromRegistry(registry registryTypes.ImageRegistry) []gcrRemote.Option {
	registryCfg := &registryTypes.Config{}
	if cfg := registry.Config(); cfg != nil {
		registryCfg = cfg
	}
	authCfg := authn.AuthConfig{
		Username: registryCfg.Username,
		Password: registryCfg.Password,
	}

	auth := authn.FromConfig(authCfg)

	// By default, the proxy will be taken from environment, assuming this will be in line with our general proxy
	// strategy.
	transport := gcrRemote.DefaultTransport
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: registryCfg.Insecure}

	return []gcrRemote.Option{
		gcrRemote.WithAuth(auth),
		gcrRemote.WithTransport(transport),
	}
}
