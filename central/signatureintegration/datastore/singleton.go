package datastore

import (
	"context"
	"encoding/pem"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once           sync.Once
	instance       DataStore
	keyUpdater     *updater
	keyUpdaterLock sync.Mutex
)

// upsertRedHatSignatureIntegration builds the Red Hat signature integration from
// the embedded key plus any keys found in the runtime directory, then upserts it
// to the datastore. Duplicate keys (by canonical PEM) are silently dropped.
// It is called at startup and as the updater's onSuccess callback.
func upsertRedHatSignatureIntegration(siStore store.SignatureIntegrationStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Start from a clone of the default integration so we never mutate the global.
	integration := signatures.DefaultRedHatSignatureIntegration.CloneVT()

	// Build a set of canonical PEM values from the embedded keys for deduplication.
	seen := make(map[string]struct{})
	for _, k := range integration.GetCosign().GetPublicKeys() {
		// Normalise the embedded PEM so comparison with dir keys is consistent.
		if block, _ := pem.Decode([]byte(k.GetPublicKeyPemEnc())); block != nil {
			seen[string(pem.EncodeToMemory(block))] = struct{}{}
		} else {
			seen[k.GetPublicKeyPemEnc()] = struct{}{}
		}
	}

	// Load additional keys from the runtime directory. The keyloader already
	// returns canonical PEM (via pem.EncodeToMemory), so string comparison is safe.
	runtimeDir := env.RedHatSigningKeysRuntimeDir.Setting()
	additionalKeys, err := loadKeysFromDir(runtimeDir)
	if err != nil {
		log.Warnf("Failed to load Red Hat signing keys from %q: %v", runtimeDir, err)
	}
	for _, k := range additionalKeys {
		if _, dup := seen[k.GetPublicKeyPemEnc()]; dup {
			log.Debugf("Skipping duplicate Red Hat signing key %q from directory", k.GetName())
			continue
		}
		seen[k.GetPublicKeyPemEnc()] = struct{}{}
		integration.Cosign.PublicKeys = append(integration.Cosign.PublicKeys, k)
	}

	log.Debugf("Upserting Red Hat signature integration %q (%s) with %d key(s)",
		integration.GetName(),
		integration.GetId(),
		len(integration.GetCosign().GetPublicKeys()),
	)
	utils.Should(errors.Wrap(siStore.Upsert(ctx, integration), "upserting Red Hat signature integration"))
}

func startRedHatSigningKeyUpdater(siStore store.SignatureIntegrationStore) {
	manifestURL := env.RedHatSigningKeyManifestURL.Setting()
	if manifestURL == "" {
		log.Info("Red Hat signing key manifest URL not configured, skipping key updater")
		return
	}

	onSuccess := func() { upsertRedHatSignatureIntegration(siStore) }

	u, err := newUpdater(
		&http.Client{
			Timeout:   30 * time.Second,
			Transport: proxy.RoundTripper(),
		},
		manifestURL,
		env.RedHatSigningKeysRuntimeDir.Setting(),
		env.RedHatSigningKeyUpdateInterval.DurationSetting(),
		onSuccess,
	)
	if err != nil {
		utils.Should(errors.Wrap(err, "creating Red Hat signing key updater"))
		return
	}

	keyUpdaterLock.Lock()
	keyUpdater = u
	keyUpdaterLock.Unlock()

	u.Start()
}

// StopRedHatSigningKeyUpdater stops the background key updater if running.
func StopRedHatSigningKeyUpdater() {
	keyUpdaterLock.Lock()
	u := keyUpdater
	keyUpdaterLock.Unlock()

	if u != nil {
		u.Stop()
	}
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		upsertRedHatSignatureIntegration(storage)
		instance = New(storage, policyDataStore.Singleton())
		startRedHatSigningKeyUpdater(storage)
	})
	return instance
}
