package datastore

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
)

const maxBundleFileSize = 5 * 1024 * 1024 // 5 MB

type keyBundleWatcher struct {
	filePath string
	interval time.Duration
	siStore  store.SignatureIntegrationStore
	stopSig  concurrency.Signal
	lastHash [sha256.Size]byte
}

const minWatchInterval = 5 * time.Second

func newKeyBundleWatcher(filePath string, interval time.Duration, siStore store.SignatureIntegrationStore) *keyBundleWatcher {
	if interval < minWatchInterval {
		log.Warnf("Watch interval %v is below minimum %v, clamping", interval, minWatchInterval)
		interval = minWatchInterval
	}
	return &keyBundleWatcher{
		filePath: filePath,
		interval: interval,
		siStore:  siStore,
		stopSig:  concurrency.NewSignal(),
	}
}

func (w *keyBundleWatcher) Start() {
	go w.run()
}

func (w *keyBundleWatcher) Stop() {
	w.stopSig.Signal()
}

func (w *keyBundleWatcher) run() {
	log.Info("Starting Red Hat signing key bundle watcher")
	defer log.Info("Stopped Red Hat signing key bundle watcher")

	w.checkAndUpsert()

	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			w.checkAndUpsert()
		case <-w.stopSig.Done():
			return
		}
	}
}

func (w *keyBundleWatcher) checkAndUpsert() {
	data, err := os.ReadFile(w.filePath)
	if errors.Is(err, os.ErrNotExist) {
		log.Debugf("Key bundle file %q does not exist, skipping", w.filePath)
		w.lastHash = [sha256.Size]byte{}
		return
	}
	if err != nil {
		log.Warnf("Failed to read key bundle file %q: %v", w.filePath, err)
		return
	}
	if len(data) > maxBundleFileSize {
		log.Warnf("Key bundle file %q exceeds maximum size of %d bytes, skipping", w.filePath, maxBundleFileSize)
		return
	}

	hash := sha256.Sum256(data)
	if hash == w.lastHash {
		log.Debug("Key bundle file unchanged, skipping")
		return
	}

	bundle, err := parseKeyBundle(data)
	if err != nil {
		log.Warnf("Invalid key bundle file %q: %v", w.filePath, err)
		w.lastHash = hash
		return
	}

	si := bundle.toSignatureIntegration()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	if err := w.siStore.Upsert(ctx, si); err != nil {
		log.Errorf("Failed to upsert Red Hat signature integration from key bundle: %v", err)
		return
	}

	w.lastHash = hash
	keyNames := make([]string, 0, len(bundle.Keys))
	for _, k := range bundle.Keys {
		keyNames = append(keyNames, k.Name)
	}
	log.Infof("Updated Red Hat signature integration with %d key(s) from bundle: [%s]",
		len(bundle.Keys), strings.Join(keyNames, ", "))
}
