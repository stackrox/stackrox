package datastore

import (
	"context"
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

func upsertDefaultRedHatSignatureIntegration(siStore store.SignatureIntegrationStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	log.Debugf("Upserting default Red Hat signature integration %q (%s)",
		signatures.DefaultRedHatSignatureIntegration.GetName(),
		signatures.DefaultRedHatSignatureIntegration.GetId(),
	)
	err := siStore.Upsert(ctx, signatures.DefaultRedHatSignatureIntegration)
	utils.Should(errors.Wrap(err, "upserting default Red Hat signature integration"))
}

func startRedHatSigningKeyUpdater() {
	manifestURL := env.RedHatSigningKeyManifestURL.Setting()
	if manifestURL == "" {
		log.Info("Red Hat signing key manifest URL not configured, skipping key updater")
		return
	}

	u, err := newUpdater(
		&http.Client{
			Timeout:   30 * time.Second,
			Transport: proxy.RoundTripper(),
		},
		manifestURL,
		env.RedHatSigningKeysRuntimeDir.Setting(),
		env.RedHatSigningKeyUpdateInterval.DurationSetting(),
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
		upsertDefaultRedHatSignatureIntegration(storage)
		instance = New(storage, policyDataStore.Singleton())
		startRedHatSigningKeyUpdater()
	})
	return instance
}
