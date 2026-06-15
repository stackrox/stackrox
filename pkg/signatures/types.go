package signatures

import (
	_ "embed"

	"github.com/stackrox/rox/generated/storage"
)

const (
	// SignatureIntegrationIDPrefix should be prepended to every human-hostile ID of a
	// signature integration for readability, e.g.,
	//
	//	"io.stackrox.signatureintegration.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	SignatureIntegrationIDPrefix = "io.stackrox.signatureintegration."

	defaultRedHatIntegrationID   = SignatureIntegrationIDPrefix + "12a37a37-760e-4388-9e79-d62726c075b2"
	defaultRedHatIntegrationName = "Red Hat"
)

// bundle.json is the canonical key bundle containing Red Hat signing keys.
// It is the single source of truth: the same file is published to GCS for
// runtime updates and embedded here for the first-install seed.
//
//go:embed "bundle.json"
var embeddedKeyBundleJSON []byte

// DefaultRedHatSignatureIntegration is the seed integration created on first
// install. Its keys come from the embedded bundle.json.
// PLEASE DON'T CHANGE THE ID!! A migration may be needed if this is changed.
var DefaultRedHatSignatureIntegration = mustParseEmbeddedBundle(embeddedKeyBundleJSON)

// BundleToSignatureIntegration converts a parsed KeyBundle into the default
// Red Hat SignatureIntegration, using the well-known ID and name.
func BundleToSignatureIntegration(kb *KeyBundle) *storage.SignatureIntegration {
	publicKeys := make([]*storage.CosignPublicKeyVerification_PublicKey, 0, len(kb.Keys))
	for _, entry := range kb.Keys {
		publicKeys = append(publicKeys, &storage.CosignPublicKeyVerification_PublicKey{
			Name:            entry.Name,
			PublicKeyPemEnc: entry.PEM,
		})
	}
	return &storage.SignatureIntegration{
		Id:   defaultRedHatIntegrationID,
		Name: defaultRedHatIntegrationName,
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: publicKeys,
		},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
}

func mustParseEmbeddedBundle(data []byte) *storage.SignatureIntegration {
	bundle, err := ParseKeyBundle(data)
	if err != nil {
		panic("embedded key bundle is invalid: " + err.Error())
	}
	return BundleToSignatureIntegration(bundle)
}
