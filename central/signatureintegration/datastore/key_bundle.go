package datastore

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/filewatcher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
)

var redHatKeyBundlePath = signatures.RedHatKeyBundlePath

// ensureKeyBundleDirectory creates the directory where the watcher and
// downloader expect to find bundle.json.
func ensureKeyBundleDirectory() {
	dir := filepath.Dir(redHatKeyBundlePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Warnf("Failed to create key bundle directory %q: %v", dir, err)
	}
}

// writeExampleBundle writes bundle.example.json so offline-mode customers can
// see the expected format without consulting docs.
func writeExampleBundle() {
	dir := filepath.Dir(redHatKeyBundlePath)
	examplePath := filepath.Join(dir, "bundle.example.json")
	if err := os.WriteFile(examplePath, signatures.DefaultBundleJSON(), 0600); err != nil {
		log.Warnf("Failed to write example bundle to %q: %v", examplePath, err)
	}
}

func keyBundleHandler(siStore store.SignatureIntegrationStore) filewatcher.Handler {
	return func(data []byte) error {
		bundle, err := signatures.ParseKeyBundle(data)
		if err != nil {
			log.Warnf("Invalid key bundle file: %v", err)
			watcherFileErrorTotal.Inc()
			return nil
		}

		si := signatures.BundleToSignatureIntegration(bundle)
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
