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
	// SchemaVersion1 is the first schema version of the key bundle format.
	SchemaVersion1 = "1.0"
)

var (
	// ErrKeyBundleEmpty is returned when the bundle contains no key groups at all.
	ErrKeyBundleEmpty = errox.InvalidArgs.New("key bundle contains no key groups")
	// ErrNoSupportedKeys is returned when the bundle contains only unrecognized key groups.
	ErrNoSupportedKeys = errox.InvalidArgs.New("key bundle contains no supported key types")

	ErrKeyNameEmpty         = errox.InvalidArgs.New("empty name")
	ErrKeyNamePathSeparator = errox.InvalidArgs.New("must not contain path separators")
	ErrKeyNameDuplicate     = errox.InvalidArgs.New("duplicate key name")
	ErrKeyInvalidPEM        = errox.InvalidArgs.New("invalid PEM-encoded public key")
	ErrUnmarshalling        = errox.InvalidArgs.New("unmarshalling key bundle JSON")

	// knownBundleFields lists JSON fields that ParseKeyBundle recognizes.
	// Any other top-level field triggers a warning about unrecognized content.
	knownBundleFields = set.NewFrozenStringSet("schemaVersion", "cosignKeys")
)

// KeyBundle represents a set of public keys in the key bundle JSON format.
// Keys are grouped by type: each supported key type has its own field with
// type-specific entry structure. Unknown key groups are logged and skipped
// for forward compatibility.
type KeyBundle struct {
	SchemaVersion string      `json:"schemaVersion,omitempty"`
	CosignKeys    []CosignKey `json:"cosignKeys,omitempty"`
}

// CosignKey is a named cosign public key within a KeyBundle.
type CosignKey struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

// ParseKeyBundle parses and validates a key bundle JSON.
//
// Schema version handling:
//   - Known versions (e.g. "1.0"): accepted.
//   - Unknown versions: accepted with a warning. The parser extracts what it
//     understands (cosignKeys, etc.) regardless of version, so older code can
//     still use bundles produced for newer schema versions.
//
// Key group handling:
//   - Known groups (cosignKeys): parsed and validated.
//   - Unknown groups: logged and skipped (forward compatibility).
//   - Returns ErrKeyBundleEmpty if the bundle has no key groups at all.
//   - Returns ErrNoSupportedKeys if the bundle only has unrecognized groups.
func ParseKeyBundle(data []byte) (*KeyBundle, error) {
	var bundle KeyBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, ErrUnmarshalling.CausedBy(err)
	}

	if bundle.SchemaVersion != SchemaVersion1 {
		log.Warnf("Key bundle has unknown schema version %q; attempting to parse with known fields", bundle.SchemaVersion)
	}

	// Detect unrecognized top-level fields for forward-compatibility diagnostics.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, ErrUnmarshalling.CausedBy(err)
	}
	hasUnknownGroups := false
	for field := range raw {
		if !knownBundleFields.Contains(field) {
			log.Warnf("Key bundle contains unrecognized field %q; skipping", field)
			hasUnknownGroups = true
		}
	}

	if len(bundle.CosignKeys) == 0 {
		if hasUnknownGroups {
			return nil, ErrNoSupportedKeys
		}
		return nil, ErrKeyBundleEmpty
	}

	seenNames := set.NewStringSet()
	for i := range bundle.CosignKeys {
		key := &bundle.CosignKeys[i]
		key.Name = strings.TrimSpace(key.Name)
		if key.Name == "" {
			return nil, ErrKeyNameEmpty.CausedByf("cosignKeys[%d]", i)
		}
		if strings.ContainsAny(key.Name, "/\\") {
			return nil, ErrKeyNamePathSeparator.CausedByf("key name %q", key.Name)
		}
		if !seenNames.Add(key.Name) {
			return nil, ErrKeyNameDuplicate.CausedByf("%q", key.Name)
		}
		keyBlock, rest := pem.Decode([]byte(strings.TrimSpace(key.PublicKey)))
		if !IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, ErrKeyInvalidPEM.CausedByf("key %q", key.Name)
		}
		key.PublicKey = string(pem.EncodeToMemory(keyBlock))
	}

	return &bundle, nil
}

// ToSignatureIntegration converts a parsed KeyBundle into the default
// Red Hat SignatureIntegration, using the well-known ID and name.
// Returns ErrNoSupportedKeys if the bundle contains no cosign keys.
func (kb *KeyBundle) ToSignatureIntegration() (*storage.SignatureIntegration, error) {
	if len(kb.CosignKeys) == 0 {
		return nil, ErrNoSupportedKeys
	}
	publicKeys := make([]*storage.CosignPublicKeyVerification_PublicKey, 0, len(kb.CosignKeys))
	for _, key := range kb.CosignKeys {
		publicKeys = append(publicKeys, &storage.CosignPublicKeyVerification_PublicKey{
			Name:            key.Name,
			PublicKeyPemEnc: key.PublicKey,
		})
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
