package datastore

import (
	"encoding/json"
	"encoding/pem"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
)

var (
	errKeyBundleEmpty          = errors.New("key bundle must contain at least one key")
	errKeyNameEmpty            = errors.New("empty name")
	errKeyNamePathSeparator    = errors.New("must not contain path separators")
	errKeyNameDuplicate        = errors.New("duplicate key name")
	errKeyInvalidPEM           = errors.New("invalid PEM-encoded public key")
)

type keyBundle struct {
	Keys []keyBundleEntry `json:"keys"`
}

type keyBundleEntry struct {
	Name string `json:"name"`
	PEM  string `json:"pem"`
}

func parseKeyBundle(data []byte) (*keyBundle, error) {
	var bundle keyBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, errors.Wrap(err, "unmarshalling key bundle JSON")
	}
	if len(bundle.Keys) == 0 {
		return nil, errKeyBundleEmpty
	}
	seenNames := make(map[string]struct{}, len(bundle.Keys))
	for i := range bundle.Keys {
		entry := &bundle.Keys[i]
		entry.Name = strings.TrimSpace(entry.Name)
		if entry.Name == "" {
			return nil, errors.Wrapf(errKeyNameEmpty, "key at index %d", i)
		}
		if strings.ContainsAny(entry.Name, "/\\") {
			return nil, errors.Wrapf(errKeyNamePathSeparator, "key name %q", entry.Name)
		}
		if _, exists := seenNames[entry.Name]; exists {
			return nil, errors.Wrapf(errKeyNameDuplicate, "%q", entry.Name)
		}
		seenNames[entry.Name] = struct{}{}
		keyBlock, rest := pem.Decode([]byte(strings.TrimSpace(entry.PEM)))
		if !signatures.IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, errors.Wrapf(errKeyInvalidPEM, "key %q", entry.Name)
		}
		entry.PEM = string(pem.EncodeToMemory(keyBlock))
	}
	return &bundle, nil
}

func (kb *keyBundle) toDefaultSignatureIntegration() *storage.SignatureIntegration {
	publicKeys := make([]*storage.CosignPublicKeyVerification_PublicKey, 0, len(kb.Keys))
	for _, entry := range kb.Keys {
		publicKeys = append(publicKeys, &storage.CosignPublicKeyVerification_PublicKey{
			Name:            entry.Name,
			PublicKeyPemEnc: entry.PEM,
		})
	}
	return &storage.SignatureIntegration{
		Id:   signatures.DefaultRedHatSignatureIntegration.GetId(),
		Name: signatures.DefaultRedHatSignatureIntegration.GetName(),
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: publicKeys,
		},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
}
