package datastore

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/filewatcher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
)

var (
	errKeyBundleEmpty       = errors.New("key bundle must contain at least one key")
	errKeyNameEmpty         = errors.New("empty name")
	errKeyNamePathSeparator = errors.New("must not contain path separators")
	errKeyNameDuplicate     = errors.New("duplicate key name")
	errKeyInvalidPEM        = errors.New("invalid PEM-encoded public key")
)

type keyBundle struct {
	Keys []keyBundleEntry `json:"keys"`
}

type keyBundleEntry struct {
	Name string `json:"name"`
	PEM  string `json:"pem"`
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

// redHatKeyBundlePath is the well-known path where the key bundle file is stored.
// The file downloader writes the bundle here; the file watcher reads it.
var redHatKeyBundlePath = filepath.Join(os.TempDir(), "redhat-signing-keys", "bundle.json")

func keyBundleHandler(siStore store.SignatureIntegrationStore) filewatcher.Handler {
	return func(data []byte) error {
		bundle, err := parseKeyBundle(data)
		if err != nil {
			log.Warnf("Invalid key bundle file: %v", err)
			watcherFileErrorTotal.Inc()
			return nil
		}

		si := bundle.toDefaultSignatureIntegration()
		ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
		if err := siStore.Upsert(ctx, si); err != nil {
			log.Errorf("Failed to upsert Red Hat signature integration from key bundle: %v", err)
			watcherUpsertTotal.WithLabelValues("error").Inc()
			return err
		}

		watcherUpsertTotal.WithLabelValues("success").Inc()
		watcherKeyCount.Set(float64(len(bundle.Keys)))
		watcherLastSuccessTimestamp.SetToCurrentTime()

		keyNames := make([]string, 0, len(bundle.Keys))
		for _, k := range bundle.Keys {
			keyNames = append(keyNames, k.Name)
		}
		log.Infof("Updated Red Hat signature integration with %d key(s) from bundle: [%s]",
			len(bundle.Keys), strings.Join(keyNames, ", "))
		return nil
	}
}

func startKeyBundleWatcher(siStore store.SignatureIntegrationStore) {
	interval := env.RedHatSigningKeyWatchInterval.DurationSetting()
	if interval == 0 {
		log.Info("Red Hat signing key bundle watcher is disabled (ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL=0)")
		return
	}

	w := filewatcher.New(redHatKeyBundlePath, interval, keyBundleHandler(siStore),
		filewatcher.WithOnError(func(_ error) {
			watcherFileErrorTotal.Inc()
		}),
	)
	w.Start()
	bundleWatcher = w
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
