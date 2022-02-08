package datastore

import (
	"context"
	"sort"
	"testing"

	"github.com/stackrox/rox/central/role/resources"
	signatureRocksdb "github.com/stackrox/rox/central/signatureintegration/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestSignatureDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(signatureDataStoreTestSuite))
}

type signatureDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	dataStore DataStore
	storage   signatureRocksdb.Store
	rocksie   *rocksdb.RocksDB
}

func (s *signatureDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.SignatureIntegration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.SignatureIntegration)))
	s.noAccessCtx = sac.WithNoAccess(context.Background())

	s.rocksie = rocksdbtest.RocksDBForT(s.T())
	var err error
	s.storage, err = signatureRocksdb.New(s.rocksie)
	s.Require().NoError(err)
	s.dataStore = New(s.storage)
}

func (s *signatureDataStoreTestSuite) TestAddSignatureIntegration() {
	// 1. Added integration can be accessed via GetSignatureIntegration
	integrationID := "addId"
	signatureIntegration := getSignatureIntegration(integrationID, "name")
	err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	integration, found, err := s.dataStore.GetSignatureIntegration(s.hasReadCtx, integrationID)
	s.True(found)
	s.NoError(err)
	s.Equal(signatureIntegration, integration)

	// 2. Cannot add new integration with already existing id
	signatureIntegration = getSignatureIntegration(integrationID, "name2")
	err = s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.Error(err)

	// 3. Need write permission to add new integration
	signatureIntegration = getSignatureIntegration(integrationID, "name2")
	err = s.dataStore.AddSignatureIntegration(s.hasReadCtx, signatureIntegration)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *signatureDataStoreTestSuite) TestUpdateSignatureIntegration() {
	integrationID := "updateId"
	signatureIntegration := getSignatureIntegration(integrationID, "name")
	err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	// 1. Modifications to integration are visible via GetSignatureIntegration
	signatureIntegration.Name = "name2"
	err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	integration, found, err := s.dataStore.GetSignatureIntegration(s.hasReadCtx, integrationID)
	s.True(found)
	s.NoError(err)
	s.Equal(signatureIntegration, integration)

	// 2. Cannot update non-existing integration
	nonExistingIntegration := getSignatureIntegration("idonotexist", "name")
	err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, nonExistingIntegration)
	s.Error(err)

	// 3. Need write permission to update integration
	err = s.dataStore.UpdateSignatureIntegration(s.hasReadCtx, signatureIntegration)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *signatureDataStoreTestSuite) TestRemoveSignatureIntegration() {
	integrationID := "removeId"
	signatureIntegration := getSignatureIntegration(integrationID, "name")
	err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	// 1. Removed integration is not accessible via GetSignatureIntegration
	err = s.dataStore.RemoveSignatureIntegration(s.hasWriteCtx, integrationID)
	s.NoError(err)
	_, found, _ := s.dataStore.GetSignatureIntegration(s.hasReadCtx, integrationID)
	s.False(found)

	// 2. Can add integration with the same id after removal
	err = s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	// 3. Need write permission to remove integration
	err = s.dataStore.RemoveSignatureIntegration(s.hasReadCtx, "nonExistentRemoveId")
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *signatureDataStoreTestSuite) TestGetAllSignatureIntegrations() {
	integrationID1 := "id1"
	signatureIntegration := getSignatureIntegration(integrationID1, "name")
	err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	integrationID2 := "id2"
	signatureIntegration = getSignatureIntegration(integrationID2, "name2")
	err = s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	// 1. All integrations are returned
	integrations, err := s.dataStore.GetAllSignatureIntegrations(s.hasReadCtx)
	s.NoError(err)
	s.Len(integrations, 2)
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].GetId() < integrations[j].GetId()
	})
	s.Equal(integrationID1, integrations[0].GetId())
	s.Equal(integrationID2, integrations[1].GetId())
	// 2. Need read permission to get signature integrations
	integrations, err = s.dataStore.GetAllSignatureIntegrations(s.noAccessCtx)
	s.NoError(err)
	s.Len(integrations, 0)
}

func (s *signatureDataStoreTestSuite) TestGetSignatureIntegration() {
	// 1. Need read permission to get signature integration
	integrationID := "getId"
	signatureIntegration := getSignatureIntegration(integrationID, "name")
	err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)

	_, found, err := s.dataStore.GetSignatureIntegration(s.noAccessCtx, integrationID)
	s.NoError(err)
	s.False(found)
}

func getSignatureIntegration(id, name string) *storage.SignatureIntegration {
	signatureIntegration := &storage.SignatureIntegration{
		Id:   id,
		Name: name,
		SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
			{
				Id: "configId",
				Config: &storage.SignatureVerificationConfig_CosignVerification{
					CosignVerification: &storage.CosignPublicKeyVerification{
						PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
							{
								Name:            "key1",
								PublicKeyPemEnc: "abrrrrrrrrrr",
							},
						},
					},
				},
			},
		},
	}
	return signatureIntegration
}

func (s *signatureDataStoreTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksie)
}
