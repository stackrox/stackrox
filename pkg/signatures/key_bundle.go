package signatures

import (
	"encoding/json"
	"encoding/pem"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

var (
	ErrKeyBundleEmpty       = errox.InvalidArgs.New("key bundle must contain at least one key")
	ErrKeyNameEmpty         = errox.InvalidArgs.New("empty name")
	ErrKeyNamePathSeparator = errox.InvalidArgs.New("must not contain path separators")
	ErrKeyNameDuplicate     = errox.InvalidArgs.New("duplicate key name")
	ErrKeyInvalidPEM        = errox.InvalidArgs.New("invalid PEM-encoded public key")
	ErrUnmarshalling        = errox.InvalidArgs.New("unmarshalling key bundle JSON")
)

// KeyBundle represents a set of public keys in the key bundle JSON format.
type KeyBundle struct {
	Keys []KeyBundleEntry `json:"keys"`
}

// KeyBundleEntry is a single named public key within a KeyBundle.
type KeyBundleEntry struct {
	Name string `json:"name"`
	PEM  string `json:"pem"`
}

// ParseKeyBundle parses and validates a key bundle JSON. All keys must be valid
// PEM-encoded public keys; if any key fails validation the entire bundle is rejected.
func ParseKeyBundle(data []byte) (*KeyBundle, error) {
	var bundle KeyBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, ErrUnmarshalling.CausedBy(err)
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
		keyBlock, rest := pem.Decode([]byte(strings.TrimSpace(entry.PEM)))
		if !IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, ErrKeyInvalidPEM.CausedByf("key %q", entry.Name)
		}
		entry.PEM = string(pem.EncodeToMemory(keyBlock))
	}
	return &bundle, nil
}

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
		Id:   DefaultRedHatIntegrationID,
		Name: DefaultRedHatIntegrationName,
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: publicKeys,
		},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
}
