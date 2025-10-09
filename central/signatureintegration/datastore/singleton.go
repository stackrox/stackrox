package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatureintegration"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
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

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		upsertDefaultRedHatSignatureIntegration(storage)
		instance = New(storage, policyDataStore.Singleton())
		getterFunc := func() signatureintegration.Getter { return instance }
		imageDS.Singleton().SetSignatureIntegrationGetterFunc(getterFunc)
		if features.FlattenImageData.Enabled() {
			imageV2DS.Singleton().SetSignatureIntegrationGetterFunc(getterFunc)
		}
	})
	return instance
}
