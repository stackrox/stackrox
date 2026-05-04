package datastore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
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
	doneSig  concurrency.Signal
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
		doneSig:  concurrency.NewSignal(),
	}
}

func (w *keyBundleWatcher) Start() {
	go w.run()
}

func (w *keyBundleWatcher) Stop() {
	w.stopSig.Signal()
	<-w.doneSig.Done()
}

func (w *keyBundleWatcher) run() {
	log.Info("Starting Red Hat signing key bundle watcher")
	defer log.Info("Stopped Red Hat signing key bundle watcher")
	defer w.doneSig.Signal()

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
	info, err := os.Stat(w.filePath)
	if errors.Is(err, os.ErrNotExist) {
		log.Debugf("Key bundle file %q does not exist, skipping", w.filePath)
		w.lastHash = [sha256.Size]byte{}
		return
	}
	if err != nil {
		log.Warnf("Failed to stat key bundle file %q: %v", w.filePath, err)
		watcherFileErrorTotal.Inc()
		return
	}
	if info.Size() > int64(maxBundleFileSize) {
		// Use a fingerprint based on size+mtime to suppress repeated warnings for the same oversized file.
		fingerprint := sha256.Sum256([]byte(fmt.Sprintf("oversize:%d:%d", info.Size(), info.ModTime().UnixNano())))
		if fingerprint != w.lastHash {
			log.Warnf("Key bundle file %q exceeds maximum size (%d bytes > %d), skipping",
				w.filePath, info.Size(), maxBundleFileSize)
			w.lastHash = fingerprint
			watcherFileErrorTotal.Inc()
		}
		return
	}

	data, err := os.ReadFile(w.filePath)
	if err != nil {
		log.Warnf("Failed to read key bundle file %q: %v", w.filePath, err)
		watcherFileErrorTotal.Inc()
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
		watcherFileErrorTotal.Inc()
		return
	}

	si := bundle.toDefaultSignatureIntegration()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	if err := w.siStore.Upsert(ctx, si); err != nil {
		log.Errorf("Failed to upsert Red Hat signature integration from key bundle: %v", err)
		watcherUpsertTotal.WithLabelValues("error").Inc()
		return
	}

	w.lastHash = hash
	watcherUpsertTotal.WithLabelValues("success").Inc()
	watcherKeyCount.Set(float64(len(bundle.Keys)))
	watcherLastSuccessTimestamp.SetToCurrentTime()

	keyNames := make([]string, 0, len(bundle.Keys))
	for _, k := range bundle.Keys {
		keyNames = append(keyNames, k.Name)
	}
	log.Infof("Updated Red Hat signature integration with %d key(s) from bundle: [%s]",
		len(bundle.Keys), strings.Join(keyNames, ", "))
}
