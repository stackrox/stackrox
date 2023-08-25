//go:build sql_integration

package datastore

import (
	"context"
	"sort"
	"testing"

	"github.com/pkg/errors"
	policyDataStoreMock "github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

	dataStore         DataStore
	db                *pgtest.TestPostgres
	storage           store.SignatureIntegrationStore
	policyStorageMock *policyDataStoreMock.MockDataStore
}

func (s *signatureDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.noAccessCtx = sac.WithNoAccess(context.Background())

	s.db = pgtest.ForT(s.T())
	var err error
	s.storage = postgres.New(s.db)
	s.Require().NoError(err)

	s.policyStorageMock = policyDataStoreMock.NewMockDataStore(gomock.NewController(s.T()))

	s.dataStore = New(s.storage, s.policyStorageMock)
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

	// Set the mock to return empty policy.
	s.policyStorageMock.EXPECT().GetAllPolicies(gomock.Any()).Return(nil, nil).AnyTimes()

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

func (s *signatureDataStoreTestSuite) TestRemoveSignatureIntegrationReferencedByPolicy() {
	signatureIntegration := newSignatureIntegration("name")
	savedIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, signatureIntegration)
	s.NoError(err)
	s.NotNil(savedIntegration)

	// 1. Return a policy that will reference the signature integration.
	s.policyStorageMock.EXPECT().GetAllPolicies(gomock.Any()).Return([]*storage.Policy{{
		Name: "policy-referencing-integration",
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: search.ImageSignatureVerifiedBy.String(),
				Values: []*storage.PolicyValue{{
					Value: savedIntegration.GetId(),
				}}}}}}}}, nil).MaxTimes(2)

	// 2. Removing the integration should fail due to the existing reference.
	err = s.dataStore.RemoveSignatureIntegration(s.hasWriteCtx, savedIntegration.GetId())
	s.Error(err)
	s.ErrorIs(err, errox.ReferencedByAnotherObject)

	// 3. Return an error when retrieving policies.
	s.policyStorageMock.EXPECT().GetAllPolicies(gomock.Any()).Return(nil, errors.New("some error"))

	// 4. Removing the integration should fail due to an error when retrieving policies.
	err = s.dataStore.RemoveSignatureIntegration(s.hasWriteCtx, savedIntegration.GetId())
	s.Error(err)

	// 5. Return a policy that does not reference the signature integration.
	s.policyStorageMock.EXPECT().GetAllPolicies(gomock.Any()).Return([]*storage.Policy{{
		Name: "policy-referencing-integration",
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: "some other field",
				Values: []*storage.PolicyValue{{
					Value: "some other value",
				}}}}}}}}, nil).MaxTimes(2)

	// 6. Removing the integration should work now and is not accessible via GetSignatureIntegration.
	err = s.dataStore.RemoveSignatureIntegration(s.hasWriteCtx, savedIntegration.GetId())
	s.NoError(err)
	_, found, _ := s.dataStore.GetSignatureIntegration(s.hasReadCtx, savedIntegration.GetId())
	s.False(found)
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
	s.db.Teardown(s.T())
}

func TestRemovePoliciesInvisibleToUser(t *testing.T) {
	cases := map[string]struct {
		policiesVisibleToUser []*storage.Policy
		policiesWithReference []string
		expectedOutput        []string
	}{
		"empty policies visible to user": {
			policiesVisibleToUser: nil,
			policiesWithReference: []string{"policyA", "policyB"},
			expectedOutput:        []string{"<hidden>"},
		},
		"policies visible to user and policies with references are equal": {
			policiesVisibleToUser: []*storage.Policy{
				{Name: "policyA"},
				{Name: "policyB"},
			},
			policiesWithReference: []string{"policyA", "policyB"},
			expectedOutput:        []string{"policyA", "policyB"},
		},
		"policies visible to user are greater than policies with references": {
			policiesVisibleToUser: []*storage.Policy{
				{Name: "policyA"},
				{Name: "policyB"},
				{Name: "policyC"},
			},
			policiesWithReference: []string{"policyA", "policyB"},
			expectedOutput:        []string{"policyA", "policyB"},
		},
		"policies visible to user are less than policies with references": {
			policiesVisibleToUser: []*storage.Policy{
				{Name: "policyA"},
			},
			policiesWithReference: []string{"policyA", "policyB"},
			expectedOutput:        []string{"policyA", "<hidden>"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			result := removePoliciesInvisibleToUser(c.policiesVisibleToUser, c.policiesWithReference)
			assert.ElementsMatch(t, c.expectedOutput, result)
		})
	}
}
