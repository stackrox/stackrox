package datastore

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/keybundle"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// redHatKeyBundlePath is the well-known path where the key bundle file is read from.
var redHatKeyBundlePath = filepath.Join(os.TempDir(), "redhat-signing-keys", "bundle.json")

var (
	once     sync.Once
	instance DataStore
)

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

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		seedRedHatDefaultSignatureIntegration(storage) // must run before watcher; bundle file takes precedence on first tick
		instance = New(storage, policyDataStore.Singleton())
		keybundle.StartWatcher(redHatKeyBundlePath, env.RedHatSigningKeyWatchInterval.DurationSetting(), storage)
	})
	return instance
}
