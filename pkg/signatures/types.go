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

	// DefaultRedHatIntegrationID is the well-known ID for the default Red Hat
	// signature integration. PLEASE DON'T CHANGE THIS ID!! A migration may be
	// needed if this is changed.
	DefaultRedHatIntegrationID = SignatureIntegrationIDPrefix + "12a37a37-760e-4388-9e79-d62726c075b2"

	// DefaultRedHatIntegrationName is the display name for the default Red Hat
	// signature integration.
	DefaultRedHatIntegrationName = "Red Hat"
)

//go:embed "bundle.json"
var defaultBundleJSON []byte

var DefaultRedHatSignatureIntegration = mustParseEmbeddedBundle()

func mustParseEmbeddedBundle() *storage.SignatureIntegration {
	bundle, err := ParseKeyBundle(defaultBundleJSON)
	if err != nil {
		panic("embedded bundle.json is invalid: " + err.Error())
	}
	return BundleToSignatureIntegration(bundle)
}
