package datastore

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/filewatcher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// redHatKeyBundlePath is the well-known path where the key bundle file is read from.
// An external process downloads the bundle and writes it to this path.
var redHatKeyBundlePath = filepath.Join(os.TempDir(), "redhat-signing-keys", "bundle.json")

// Stoppable represents a background process that can be stopped.
type Stoppable interface {
	Stop()
}

var (
	once     sync.Once
	instance DataStore

	bundleWatcher Stoppable
)

// KeyBundleWatcher returns the key bundle watcher for shutdown registration.
// Must only be called after Singleton().
func KeyBundleWatcher() Stoppable {
	return bundleWatcher
}

func seedRedHatDefaultSignatureIntegration(siStore store.SignatureIntegrationStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	id := signatures.DefaultRedHatSignatureIntegration.GetId()
	_, exists, err := siStore.Get(ctx, id)
	if err != nil {
		utils.Should(errors.Wrap(err, "checking for default Red Hat signature integration"))
		return
	}
	if exists {
		log.Debugf("Default Red Hat signature integration %q already exists, skipping seed", id)
		return
	}

	log.Infof("Seeding default Red Hat signature integration %q", id)
	err = siStore.Upsert(ctx, signatures.DefaultRedHatSignatureIntegration)
	utils.Should(errors.Wrap(err, "seeding default Red Hat signature integration"))
}

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

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		seedRedHatDefaultSignatureIntegration(storage) // must run before watcher; bundle file takes precedence on first tick
		instance = New(storage, policyDataStore.Singleton())
		startKeyBundleWatcher(storage)
	})
	return instance
}
