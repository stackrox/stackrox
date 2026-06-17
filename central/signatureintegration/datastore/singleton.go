package datastore

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

// Stoppable represents a background process that can be stopped.
type Stoppable interface {
	Stop()
}

var (
	once     sync.Once
	instance DataStore

	bundleUpdater Stoppable
	bundleWatcher Stoppable
)

// KeyBundleUpdater returns the key bundle updater for shutdown registration.
// Returns nil if the updater was not started (e.g. no URL configured).
// Must only be called after Singleton().
func KeyBundleUpdater() Stoppable {
	return bundleUpdater
}

// KeyBundleWatcher returns the key bundle watcher for shutdown registration.
// Must only be called after Singleton().
func KeyBundleWatcher() Stoppable {
	return bundleWatcher
}

func seedRedHatDefaultSignatureIntegration(siStore store.SignatureIntegrationStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	id := signatures.DefaultRedHatIntegrationID
	_, exists, err := siStore.Get(ctx, id)
	if err != nil {
		utils.Should(errors.Wrap(err, "checking for default Red Hat signature integration"))
		return
	}
	if exists {
		log.Debugf("Default Red Hat signature integration %q already exists, skipping seed", id)
		return
	}

	data, err := os.ReadFile(signatures.RedHatKeyBundlePath)
	if err != nil {
		utils.Should(errors.Wrapf(err, "reading key bundle file %q for initial seed", signatures.RedHatKeyBundlePath))
		return
	}
	bundle, err := signatures.ParseKeyBundle(data)
	if err != nil {
		utils.Should(errors.Wrapf(err, "parsing key bundle file %q for initial seed", signatures.RedHatKeyBundlePath))
		return
	}

	si := signatures.BundleToSignatureIntegration(bundle)
	log.Infof("Seeding default Red Hat signature integration %q with %d key(s)", id, len(bundle.Keys))
	err = siStore.Upsert(ctx, si)
	utils.Should(errors.Wrap(err, "seeding default Red Hat signature integration"))
}

func startKeyBundleUpdater() {
	if env.OfflineModeEnv.BooleanSetting() {
		log.Infof("Offline mode detected: The Red Hat signing key bundle will not be downloaded automatically. "+
			"Manual updates are possible by mounting the bundle to %q", signatures.RedHatKeyBundlePath)
		return
	}

	rawURL := env.RedHatSigningKeyBundleURL.Setting()
	if rawURL == "" {
		log.Info("ROX_REDHAT_SIGNING_KEY_BUNDLE_URL not set, key bundle updater will not start")
		return
	}
	bundleURL := urlfmt.FormatURL(rawURL, urlfmt.HTTPS, urlfmt.HonorInputSlash)

	interval := env.RedHatSigningKeyUpdateInterval.DurationSetting()

	u := filedownloader.New(bundleURL, signatures.RedHatKeyBundlePath, interval,
		filedownloader.WithOnComplete(func(err error, duration time.Duration) {
			updaterDownloadDuration.Observe(duration.Seconds())
			if err != nil {
				log.Warnf("Failed to download Red Hat signing key bundle from %q: %v", bundleURL, err)
				updaterDownloadsTotal.WithLabelValues("error").Inc()
			} else {
				updaterDownloadsTotal.WithLabelValues("success").Inc()
				updaterLastSuccessTimestamp.SetToCurrentTime()
			}
		}),
	)
	u.Start()
	bundleUpdater = u
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		seedRedHatDefaultSignatureIntegration(storage) // must run before watcher; bundle file takes precedence on first tick
		instance = New(storage, policyDataStore.Singleton())
		startKeyBundleWatcher(storage)
		startKeyBundleUpdater()
	})
	return instance
}
