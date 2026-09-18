package datastore

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/filewatcher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
)

var redHatKeyBundlePath = signatures.RedHatKeyBundlePath()

// State tracked for the key bundle watcher's StatusAdapter (debug endpoint
// observability only; the Prometheus metrics above serve monitoring).
var (
	watcherLastSuccessUnixNano atomic.Int64
	watcherLastKeyCount        atomic.Int64
	watcherErrorCount          atomic.Int64
)

func keyBundleHandler(siStore store.SignatureIntegrationStore) filewatcher.Handler {
	return func(data []byte) error {
		bundle, err := signatures.ParseKeyBundle(data)
		if err != nil {
			log.Warnf("Invalid key bundle file: %v", err)
			watcherFileErrorTotal.Inc()
			watcherErrorCount.Add(1)
			return nil
		}

		si, err := bundle.ToSignatureIntegration()
		if err != nil {
			log.Warnf("Failed to create Red Hat signature integration from key bundle: %v", err)
			watcherFileErrorTotal.Inc()
			watcherErrorCount.Add(1)
			return nil
		}
		ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
		if err := siStore.Upsert(ctx, si); err != nil {
			log.Errorf("Failed to upsert Red Hat signature integration from key bundle: %v", err)
			watcherUpsertTotal.WithLabelValues("error").Inc()
			watcherErrorCount.Add(1)
			return err
		}

		watcherUpsertTotal.WithLabelValues("success").Inc()
		cosignKeys := si.GetCosign().GetPublicKeys()
		watcherKeyCount.Set(float64(len(cosignKeys)))
		watcherLastSuccessTimestamp.SetToCurrentTime()
		watcherLastSuccessUnixNano.Store(time.Now().UnixNano())
		watcherLastKeyCount.Store(int64(len(cosignKeys)))

		keyNames := make([]string, 0, len(cosignKeys))
		for _, k := range cosignKeys {
			keyNames = append(keyNames, k.GetName())
		}
		log.Infof("Updated Red Hat signature integration with %d key(s) from bundle: [%s]",
			len(cosignKeys), strings.Join(keyNames, ", "))
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
			watcherErrorCount.Add(1)
		}),
	)
	w.Start()
	bundleWatcher = w

	// Register with the background worker registry for debug endpoint
	// observability. The watcher does not fit a standard archetype: it wraps
	// pkg/filewatcher's own filesystem polling loop rather than owning one
	// itself.
	backgroundworker.Global.Register(&backgroundworker.StatusAdapter{
		WorkerName: "key-bundle-watcher",
		WorkerKind: "file-watcher",
		StatusFunc: func() backgroundworker.WorkerStatus {
			extra := map[string]any{
				"watch_path":  redHatKeyBundlePath,
				"interval":    interval.String(),
				"key_count":   watcherLastKeyCount.Load(),
				"error_count": watcherErrorCount.Load(),
			}
			if last := watcherLastSuccessUnixNano.Load(); last != 0 {
				extra["last_success_time"] = time.Unix(0, last).UTC().Format(time.RFC3339)
			}
			return backgroundworker.WorkerStatus{
				State: "running",
				Extra: extra,
			}
		},
	})
}
