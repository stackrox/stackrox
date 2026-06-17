package signatures

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

// RedHatKeyBundlePath is the well-known path where the key bundle file is stored.
// The file is baked into the image at build time and updated at runtime by the
// downloader. The watcher polls it for changes. In offline mode, users can mount
// an updated bundle at this path.
var RedHatKeyBundlePath = "/run/stackrox.io/redhat-signing-keys/bundle.json"
