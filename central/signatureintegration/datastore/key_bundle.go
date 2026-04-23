package datastore

import (
	"encoding/json"
	"encoding/pem"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
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
		return nil, errors.New("key bundle must contain at least one key")
	}
	seenNames := make(map[string]struct{}, len(bundle.Keys))
	for i := range bundle.Keys {
		entry := &bundle.Keys[i]
		if entry.Name == "" {
			return nil, errors.Errorf("key at index %d has empty name", i)
		}
		if strings.ContainsAny(entry.Name, "/\\") {
			return nil, errors.Errorf("key name %q must not contain path separators", entry.Name)
		}
		if _, exists := seenNames[entry.Name]; exists {
			return nil, errors.Errorf("duplicate key name %q", entry.Name)
		}
		seenNames[entry.Name] = struct{}{}
		keyBlock, rest := pem.Decode([]byte(strings.TrimSpace(entry.PEM)))
		if !signatures.IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, errors.Errorf("key %q has invalid PEM-encoded public key", entry.Name)
		}
		entry.PEM = string(pem.EncodeToMemory(keyBlock))
	}
	return &bundle, nil
}

func (kb *keyBundle) toSignatureIntegration() *storage.SignatureIntegration {
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
