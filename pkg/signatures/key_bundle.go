package signatures

import (
	"encoding/json"
	"encoding/pem"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

var (
	ErrKeyBundleEmpty       = errors.New("key bundle must contain at least one key")
	ErrKeyNameEmpty         = errors.New("empty name")
	ErrKeyNamePathSeparator = errors.New("must not contain path separators")
	ErrKeyNameDuplicate     = errors.New("duplicate key name")
	ErrKeyInvalidPEM        = errors.New("invalid PEM-encoded public key")
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
		return nil, errors.Wrap(err, "unmarshalling key bundle JSON")
	}
	if len(bundle.Keys) == 0 {
		return nil, ErrKeyBundleEmpty
	}
	seenNames := make(map[string]struct{}, len(bundle.Keys))
	for i := range bundle.Keys {
		entry := &bundle.Keys[i]
		entry.Name = strings.TrimSpace(entry.Name)
		if entry.Name == "" {
			return nil, errors.Wrapf(ErrKeyNameEmpty, "key at index %d", i)
		}
		if strings.ContainsAny(entry.Name, "/\\") {
			return nil, errors.Wrapf(ErrKeyNamePathSeparator, "key name %q", entry.Name)
		}
		if _, exists := seenNames[entry.Name]; exists {
			return nil, errors.Wrapf(ErrKeyNameDuplicate, "%q", entry.Name)
		}
		seenNames[entry.Name] = struct{}{}
		keyBlock, rest := pem.Decode([]byte(strings.TrimSpace(entry.PEM)))
		if !IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, errors.Wrapf(ErrKeyInvalidPEM, "key %q", entry.Name)
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
