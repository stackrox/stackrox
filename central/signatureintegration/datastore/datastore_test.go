package datastore

import (
	"context"
	"sort"
	"testing"

	signatureRocksdb "github.com/stackrox/rox/central/signatureintegration/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errox"
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
	integration := newSignatureIntegration("name")
	savedIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, integration)
	s.NoError(err)
	s.NotNil(savedIntegration)

	acquiredIntegration, found, err := s.dataStore.GetSignatureIntegration(s.hasReadCtx, savedIntegration.GetId())
	s.True(found)
	s.NoError(err)
	s.Equal(savedIntegration, acquiredIntegration)

	// 2. Name should be unique
	integration = newSignatureIntegration("name")
	savedIntegration, err = s.dataStore.AddSignatureIntegration(s.hasWriteCtx, integration)
	s.ErrorIs(err, errox.AlreadyExists)
	s.Nil(savedIntegration)

	// 3. ID should be absent
	integration.Id = GenerateSignatureIntegrationID()
	savedIntegration, err = s.dataStore.AddSignatureIntegration(s.hasWriteCtx, integration)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(savedIntegration)

	// 4. Need write permission to add new integration
	integration = newSignatureIntegration("name2")
	savedIntegration, err = s.dataStore.AddSignatureIntegration(s.hasReadCtx, integration)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.Nil(savedIntegration)
}

func (s *signatureDataStoreTestSuite) TestUpdateSignatureIntegration() {
	signatureIntegration := newSignatureIntegration("name")
	savedIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)
	s.NotNil(savedIntegration)

	// 1. Modifications to integration are visible via GetSignatureIntegration
	savedIntegration.Name = "name2"
	hasUpdatedKeys, err := s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.False(hasUpdatedKeys)

	acquiredIntegration, found, err := s.dataStore.GetSignatureIntegration(s.hasReadCtx, savedIntegration.GetId())
	s.True(found)
	s.NoError(err)
	s.Equal(savedIntegration, acquiredIntegration)

	// 2. Cannot update non-existing integration
	nonExistingIntegration := newSignatureIntegration("idonotexist")
	hasUpdatedKeys, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, nonExistingIntegration)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
	s.False(hasUpdatedKeys)

	// 3. Need write permission to update integration
	hasUpdatedKeys, err = s.dataStore.UpdateSignatureIntegration(s.hasReadCtx, signatureIntegration)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(hasUpdatedKeys)

	// 4. Signal updated keys when keys differ
	savedIntegration.GetCosign().GetPublicKeys()[0].PublicKeyPemEnc = `-----BEGIN PUBLIC KEY-----
MEkwEwYHKoZIzj0CAQYIKoZIzj0DAQMDMgAE+Y+qPqI3geo2hQH8eK7Rn+YWG09T
ejZ5QFoj9fmxFrUyYhFap6XmTdJtEi8myBmW
-----END PUBLIC KEY-----`
	hasUpdatedKeys, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.True(hasUpdatedKeys)

	// 5. Don't signal updated keys when keys are the same
	hasUpdatedKeys, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.False(hasUpdatedKeys)

	// 6. Don't signal updated keys when only the name is changed
	savedIntegration.GetCosign().GetPublicKeys()[0].Name = "rename of public key"
	hasUpdatedKeys, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.False(hasUpdatedKeys)
}

func (s *signatureDataStoreTestSuite) TestRemoveSignatureIntegration() {
	signatureIntegration := newSignatureIntegration("name")
	savedIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)
	s.NotNil(savedIntegration)

	// 1. Removed integration is not accessible via GetSignatureIntegration
	err = s.dataStore.RemoveSignatureIntegration(s.hasWriteCtx, savedIntegration.GetId())
	s.NoError(err)
	_, found, _ := s.dataStore.GetSignatureIntegration(s.hasReadCtx, savedIntegration.GetId())
	s.False(found)

	// 2. Need write permission to remove integration
	err = s.dataStore.RemoveSignatureIntegration(s.hasReadCtx, "nonExistentRemoveId")
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	// 3. Removing non-existent id should return NotFound
	err = s.dataStore.RemoveSignatureIntegration(s.hasWriteCtx, "nonExistentRemoveId")
	s.ErrorIs(err, errox.NotFound)
}

func (s *signatureDataStoreTestSuite) TestGetAllSignatureIntegrations() {
	signatureIntegration := newSignatureIntegration("name1")
	savedIntegration1, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)
	s.NotNil(savedIntegration1)

	signatureIntegration = newSignatureIntegration("name2")
	savedIntegration2, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)
	s.NotNil(savedIntegration2)

	// 1. All integrations are returned
	integrations, err := s.dataStore.GetAllSignatureIntegrations(s.hasReadCtx)
	s.NoError(err)
	s.Len(integrations, 2)
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].GetName() < integrations[j].GetName()
	})
	s.Equal(savedIntegration1.GetId(), integrations[0].GetId())
	s.Equal(savedIntegration2.GetId(), integrations[1].GetId())

	// 2. Need read permission to get signature integrations
	integrations, err = s.dataStore.GetAllSignatureIntegrations(s.noAccessCtx)
	s.NoError(err)
	s.Len(integrations, 0)
}

func (s *signatureDataStoreTestSuite) TestGetSignatureIntegration() {
	signatureIntegration := newSignatureIntegration("name")
	savedIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)
	s.NotNil(savedIntegration)

	// 1. Need read permission to get signature integration
	result, found, err := s.dataStore.GetSignatureIntegration(s.noAccessCtx, savedIntegration.GetId())
	s.NoError(err)
	s.False(found)
	s.Nil(result)
}

func newSignatureIntegration(name string) *storage.SignatureIntegration {
	signatureIntegration := &storage.SignatureIntegration{
		Name: name,
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{
					Name:            "key1",
					PublicKeyPemEnc: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----",
				},
			},
		},
	}
	return signatureIntegration
}

func (s *signatureDataStoreTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksie)
}
