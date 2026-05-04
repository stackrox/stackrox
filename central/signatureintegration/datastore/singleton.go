package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

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

func startKeyBundleWatcher(siStore store.SignatureIntegrationStore) {
	if env.DisableRedHatSigningKeyBundleWatcher.BooleanSetting() {
		log.Info("Red Hat signing key bundle watcher is disabled via ROX_DISABLE_REDHAT_SIGNING_KEY_BUNDLE_WATCHER")
		return
	}

	filePath := env.RedHatSigningKeyBundlePath.Setting()
	interval := env.RedHatSigningKeyWatchInterval.DurationSetting()

	w := newKeyBundleWatcher(filePath, interval, siStore)
	w.Start()
	bundleWatcher = w
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		seedRedHatDefaultSignatureIntegration(storage)
		instance = New(storage, policyDataStore.Singleton())
		startKeyBundleWatcher(storage)
	})
	return instance
}
