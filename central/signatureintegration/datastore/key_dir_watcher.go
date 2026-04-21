package datastore

import (
	"context"

	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
)

// keyDirHandler implements k8scfgwatch.Handler to trigger a DB upsert of the
// Red Hat signature integration whenever the runtime signing-key directory
// changes on disk. The updater writes files there; this handler reacts to those
// writes without any direct coupling to the updater.
type keyDirHandler struct {
	siStore store.SignatureIntegrationStore
}

// OnChange satisfies the k8scfgwatch.Handler interface. Side-effect-free as
// required by the interface contract; the actual work is in OnStableUpdate.
func (h *keyDirHandler) OnChange(_ string) (interface{}, error) {
	return nil, nil
}

// OnStableUpdate is called after the watched directory has stabilised. It
// re-reads the directory and upserts the merged key set into the DB.
func (h *keyDirHandler) OnStableUpdate(_ interface{}, err error) {
	if err != nil {
		log.Warnf("Error reading Red Hat signing key directory: %v", err)
		return
	}
	upsertRedHatSignatureIntegration(h.siStore)
}

// OnWatchError is called when the directory cannot be read (e.g. it does not
// exist yet, before the first updater run). Errors are deduplicated by the
// DeduplicateWatchErrors wrapper so each distinct message is logged once.
func (h *keyDirHandler) OnWatchError(err error) {
	log.Debugf("Cannot watch Red Hat signing key directory: %v", err)
}

func startRedHatSigningKeyDirWatcher(siStore store.SignatureIntegrationStore) {
	targetDir := env.RedHatSigningKeysRuntimeDir.Setting()
	handler := k8scfgwatch.DeduplicateWatchErrors(&keyDirHandler{siStore: siStore})
	opts := k8scfgwatch.Options{
		Interval: env.RedHatSigningKeyWatchInterval.DurationSetting(),
		Force:    true, // start even before the updater creates the directory
	}
	// Errors are unreachable with Force=true, but discard defensively.
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), targetDir, handler, opts)
}
