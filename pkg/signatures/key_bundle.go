package signatures

import (
	"encoding/json"
	"encoding/pem"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

const (
	SchemaVersion1 = "1.0"
	KeyTypeCosign  = "cosign"
)

var (
	ErrKeyBundleEmpty       = errox.InvalidArgs.New("key bundle must contain at least one key")
	ErrKeyNameEmpty         = errox.InvalidArgs.New("empty name")
	ErrKeyNamePathSeparator = errox.InvalidArgs.New("must not contain path separators")
	ErrKeyNameDuplicate     = errox.InvalidArgs.New("duplicate key name")
	ErrKeyInvalidPEM        = errox.InvalidArgs.New("invalid PEM-encoded public key")
	ErrUnmarshalling        = errox.InvalidArgs.New("unmarshalling key bundle JSON")
	ErrNoSupportedKeys      = errox.InvalidArgs.New("key bundle contains no supported key types")
)

// KeyBundle represents a set of public keys in the key bundle JSON format.
type KeyBundle struct {
	SchemaVersion string           `json:"schemaVersion,omitempty"`
	Keys          []KeyBundleEntry `json:"keys"`
}

// KeyBundleEntry is a single named public key within a KeyBundle.
type KeyBundleEntry struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	PEM  string `json:"pem"`
}

// ParseKeyBundle parses and validates a key bundle JSON.
//
// Schema version handling:
//   - Known versions (e.g. "1.0"): accepted.
//   - Unknown versions: accepted with a warning. The parser extracts what it
//     understands (keys with name/type/pem) regardless of version, so older
//     code can still use bundles produced for newer schema versions.
//
// Key type handling:
//   - All types are accepted; unsupported types are filtered by ToSignatureIntegration.
//   - Keys with supported types are validated (e.g. PEM encoding for cosign keys).
//   - Keys with unsupported types are stored as-is without validation.
func ParseKeyBundle(data []byte) (*KeyBundle, error) {
	var bundle KeyBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, ErrUnmarshalling.CausedBy(err)
	}

	if bundle.SchemaVersion != "" && bundle.SchemaVersion != SchemaVersion1 {
		log.Warnf("Key bundle has unknown schema version %q; attempting to parse with known fields", bundle.SchemaVersion)
	}

	if len(bundle.Keys) == 0 {
		return nil, ErrKeyBundleEmpty
	}

	seenNames := set.NewStringSet()
	for i := range bundle.Keys {
		entry := &bundle.Keys[i]
		entry.Name = strings.TrimSpace(entry.Name)
		if entry.Name == "" {
			return nil, ErrKeyNameEmpty.CausedByf("key at index %d", i)
		}
		if strings.ContainsAny(entry.Name, "/\\") {
			return nil, ErrKeyNamePathSeparator.CausedByf("key name %q", entry.Name)
		}
		if !seenNames.Add(entry.Name) {
			return nil, ErrKeyNameDuplicate.CausedByf("%q", entry.Name)
		}
		if supportedKeyTypes.Contains(entry.Type) {
			keyBlock, rest := pem.Decode([]byte(strings.TrimSpace(entry.PEM)))
			if !IsValidPublicKeyPEMBlock(keyBlock, rest) {
				return nil, ErrKeyInvalidPEM.CausedByf("key %q", entry.Name)
			}
			entry.PEM = string(pem.EncodeToMemory(keyBlock))
		}
	}

	return &bundle, nil
}

// supportedKeyTypes defines which key types are currently handled.
var supportedKeyTypes = set.NewFrozenStringSet(KeyTypeCosign)

// ToSignatureIntegration converts a parsed KeyBundle into the default
// Red Hat SignatureIntegration, using the well-known ID and name.
// Only keys with supported types are included; unsupported key types are skipped with a warning.
// Returns ErrNoSupportedKeys if the bundle contains no keys with a supported type.
func (kb *KeyBundle) ToSignatureIntegration() (*storage.SignatureIntegration, error) {
	var publicKeys []*storage.CosignPublicKeyVerification_PublicKey
	for _, entry := range kb.Keys {
		if !supportedKeyTypes.Contains(entry.Type) {
			log.Warnf("Skipping key %q with unsupported type %q", entry.Name, entry.Type)
			continue
		}
		publicKeys = append(publicKeys, &storage.CosignPublicKeyVerification_PublicKey{
			Name:            entry.Name,
			PublicKeyPemEnc: entry.PEM,
		})
	}
	if len(publicKeys) == 0 {
		return nil, ErrNoSupportedKeys
	}
	return &storage.SignatureIntegration{
		Id:   DefaultRedHatIntegrationID,
		Name: DefaultRedHatIntegrationName,
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: publicKeys,
		},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}, nil
}
