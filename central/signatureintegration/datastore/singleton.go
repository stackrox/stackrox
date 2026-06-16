package datastore

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
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

// seedKeyBundleFile writes the embedded key data to the bundle file path if the
// file does not already exist. This gives the watcher something to poll from the
// start and provides offline-mode users with a template they can inspect and replace.
func seedKeyBundleFile() {
	bundlePath := redHatKeyBundlePath()
	if _, err := os.Stat(bundlePath); err == nil {
		log.Debugf("Key bundle file %q already exists, skipping seed", bundlePath)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Warnf("Cannot stat key bundle path %q (read-only mount?): %v, skipping seed", bundlePath, err)
		return
	}

	si := signatures.DefaultRedHatSignatureIntegration
	bundle := keyBundle{
		Keys: make([]keyBundleEntry, 0, len(si.GetCosign().GetPublicKeys())),
	}
	for _, key := range si.GetCosign().GetPublicKeys() {
		pemStr := key.GetPublicKeyPemEnc()
		if block, _ := pem.Decode([]byte(strings.TrimSpace(pemStr))); block != nil {
			pemStr = string(pem.EncodeToMemory(block))
		}
		bundle.Keys = append(bundle.Keys, keyBundleEntry{
			Name: key.GetName(),
			PEM:  pemStr,
		})
	}

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		log.Errorf("Failed to marshal embedded key bundle: %v", err)
		return
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(bundlePath), 0700); err != nil {
		log.Errorf("Failed to create key bundle directory: %v", err)
		return
	}
	if err := os.WriteFile(bundlePath, data, 0600); err != nil {
		log.Errorf("Failed to write key bundle file %q: %v", bundlePath, err)
		return
	}
	log.Infof("Seeded key bundle file %q with %d embedded key(s)", bundlePath, len(bundle.Keys))
}

func startKeyBundleUpdater() {
	if env.OfflineModeEnv.BooleanSetting() {
		log.Infof("Offline mode detected: The Red Hat signing key bundle will not be downloaded automatically. "+
			"Manual updates are possible by mounting the bundle to %q (configurable via %s)",
			redHatKeyBundlePath(), env.RedHatSigningKeyBundleFilePath.EnvVar())
		return
	}

	rawURL := env.RedHatSigningKeyBundleURL.Setting()
	if rawURL == "" {
		log.Info("ROX_REDHAT_SIGNING_KEY_BUNDLE_URL not set, key bundle updater will not start")
		return
	}
	bundleURL := urlfmt.FormatURL(rawURL, urlfmt.HTTPS, urlfmt.HonorInputSlash)

	interval := env.RedHatSigningKeyUpdateInterval.DurationSetting()

	u := filedownloader.New(bundleURL, redHatKeyBundlePath(), interval,
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
		seedKeyBundleFile()
		instance = New(storage, policyDataStore.Singleton())
		startKeyBundleWatcher(storage)
		startKeyBundleUpdater()
	})
	return instance
}
