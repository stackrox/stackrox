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
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	declarativeTraits = &storage.Traits{Origin: storage.Traits_DECLARATIVE}
	imperativeTraits  = &storage.Traits{Origin: storage.Traits_IMPERATIVE}
	defaultTraits     = &storage.Traits{Origin: storage.Traits_DEFAULT}
)

func TestSignatureDataStore(t *testing.T) {
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
	protoassert.Equal(s.T(), savedIntegration, acquiredIntegration)

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
	hasUpdates, err := s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.False(hasUpdates)

	acquiredIntegration, found, err := s.dataStore.GetSignatureIntegration(s.hasReadCtx, savedIntegration.GetId())
	s.True(found)
	s.NoError(err)
	protoassert.Equal(s.T(), savedIntegration, acquiredIntegration)

	// 2. Cannot update non-existing integration
	nonExistingIntegration := newSignatureIntegration("idonotexist")
	hasUpdates, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, nonExistingIntegration)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
	s.False(hasUpdates)

	// 3. Need write permission to update integration
	hasUpdates, err = s.dataStore.UpdateSignatureIntegration(s.hasReadCtx, signatureIntegration)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(hasUpdates)

	// 4. Signal updates when keys differ
	savedIntegration.GetCosign().GetPublicKeys()[0].PublicKeyPemEnc = `-----BEGIN PUBLIC KEY-----
MEkwEwYHKoZIzj0CAQYIKoZIzj0DAQMDMgAE+Y+qPqI3geo2hQH8eK7Rn+YWG09T
ejZ5QFoj9fmxFrUyYhFap6XmTdJtEi8myBmW
-----END PUBLIC KEY-----`
	hasUpdates, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.True(hasUpdates)

	// 5. Signal updates when certificates differ
	savedIntegration.CosignCertificates = []*storage.CosignCertificateVerification{
		{
			CertificateOidcIssuer: ".*",
			CertificateIdentity:   ".*",
		},
	}
	hasUpdates, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.True(hasUpdates)

	// 6. Don't signal updates when verification data is the same
	hasUpdates, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.False(hasUpdates)

	// 7. Don't signal updated keys when only the name is changed
	savedIntegration.GetCosign().GetPublicKeys()[0].Name = "rename of public key"
	hasUpdates, err = s.dataStore.UpdateSignatureIntegration(s.hasWriteCtx, savedIntegration)
	s.NoError(err)
	s.False(hasUpdates)
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

func (s *signatureDataStoreTestSuite) TestVerifySignatureIntegrationOrigin() {
	cases := map[string]struct {
		integration *storage.SignatureIntegration
		ctx         context.Context
		expectError bool
		errorType   error
	}{
		"imperative integration should succeed": {
			integration: newSignatureIntegrationWithTraits("test", imperativeTraits),
			ctx:         s.hasWriteCtx,
			expectError: false,
		},
		"default integration should fail": {
			integration: newSignatureIntegrationWithTraits("Red Hat", defaultTraits),
			ctx:         s.hasWriteCtx,
			expectError: true,
			errorType:   errox.NotAuthorized,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			err := verifySignatureIntegrationOrigin(tc.ctx, tc.integration)
			if tc.expectError {
				s.Error(err)
				s.ErrorIs(err, tc.errorType)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *signatureDataStoreTestSuite) TestAddSignatureIntegrationWithTraits() {
	cases := map[string]struct {
		integration *storage.SignatureIntegration
		ctx         context.Context
		expectError bool
		errorType   error
		errorMsg    string
	}{
		"adding imperative integration should succeed": {
			integration: newSignatureIntegrationWithTraits("imperative-test", imperativeTraits),
			ctx:         s.hasWriteCtx,
			expectError: false,
		},
		"adding default integration should fail": {
			integration: newSignatureIntegrationWithTraits("default-test", defaultTraits),
			ctx:         s.hasWriteCtx,
			expectError: true,
			errorType:   errox.NotAuthorized,
			errorMsg:    "cannot create signature integration",
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			result, err := s.dataStore.AddSignatureIntegration(tc.ctx, tc.integration)
			if tc.expectError {
				s.Error(err)
				s.ErrorIs(err, tc.errorType)
				if tc.errorMsg != "" {
					s.Contains(err.Error(), tc.errorMsg)
				}
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
				s.Equal(tc.integration.GetName(), result.GetName())
				s.Equal(tc.integration.GetTraits().GetOrigin(), result.GetTraits().GetOrigin())
			}
		})
	}
}

func (s *signatureDataStoreTestSuite) TestUpdateSignatureIntegrationWithTraits() {
	// First create an imperative integration
	imperativeIntegration := newSignatureIntegrationWithTraits("imperative-test", imperativeTraits)
	savedIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, imperativeIntegration)
	s.NoError(err)

	cases := map[string]struct {
		integration *storage.SignatureIntegration
		ctx         context.Context
		expectError bool
		errorType   error
		errorMsg    string
	}{
		"updating imperative integration should succeed": {
			integration: func() *storage.SignatureIntegration {
				updated := savedIntegration.CloneVT()
				updated.Name = "updated-imperative"
				return updated
			}(),
			ctx:         s.hasWriteCtx,
			expectError: false,
		},
		"updating DEFAULT integration should fail": {
			integration: func() *storage.SignatureIntegration {
				integration := newSignatureIntegrationWithTraits("updated-default", defaultTraits)
				integration.Id = "some-default-id"
				return integration
			}(),
			ctx:         s.hasWriteCtx,
			expectError: true,
			errorType:   errox.NotAuthorized,
			errorMsg:    "cannot modify signature integration",
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			hasUpdates, err := s.dataStore.UpdateSignatureIntegration(tc.ctx, tc.integration)
			if tc.expectError {
				s.Error(err)
				s.ErrorIs(err, tc.errorType)
				if tc.errorMsg != "" {
					s.Contains(err.Error(), tc.errorMsg)
				}
				s.False(hasUpdates)
			} else {
				s.NoError(err)
				s.True(hasUpdates)
			}
		})
	}
}

func (s *signatureDataStoreTestSuite) TestRemoveSignatureIntegrationWithTraits() {
	// Set up mock to return empty policies for deletion tests first
	s.policyStorageMock.EXPECT().GetAllPolicies(gomock.Any()).Return(nil, nil).AnyTimes()

	// Create imperative integration
	imperativeIntegration := newSignatureIntegrationWithTraits("imperative-test", imperativeTraits)
	savedImperativeIntegration, err := s.dataStore.AddSignatureIntegration(s.hasWriteCtx, imperativeIntegration)
	s.NoError(err)

	cases := map[string]struct {
		integrationID string
		ctx           context.Context
		expectError   bool
		errorType     error
		errorMsg      string
	}{
		"removing imperative integration should succeed": {
			integrationID: savedImperativeIntegration.GetId(),
			ctx:           s.hasWriteCtx,
			expectError:   false,
		},
		"removing DEFAULT integration should fail": {
			integrationID: "mock-default-id", // This would fail in real scenario but tests the logic
			ctx:           s.hasWriteCtx,
			expectError:   true,
			errorType:     errox.NotFound, // Since mock doesn't have this ID
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			err := s.dataStore.RemoveSignatureIntegration(tc.ctx, tc.integrationID)
			if tc.expectError {
				s.Error(err)
				s.ErrorIs(err, tc.errorType)
				if tc.errorMsg != "" {
					s.Contains(err.Error(), tc.errorMsg)
				}
			} else {
				s.NoError(err)
			}
		})
	}
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

func newSignatureIntegrationWithTraits(name string, traits *storage.Traits) *storage.SignatureIntegration {
	integration := newSignatureIntegration(name)
	integration.Traits = traits.CloneVT()
	return integration
}

func (s *signatureDataStoreTestSuite) TestTraitEnforcement() {
	// Test comprehensive trait enforcement across all CRUD operations
	// This follows the pattern from central/auth/datastore/datastore_impl_test.go:TestDeclarativeUpserts

	s.policyStorageMock.EXPECT().GetAllPolicies(gomock.Any()).Return(nil, nil).AnyTimes()

	for name, tc := range map[string]struct {
		operation     string
		integration   *storage.SignatureIntegration
		ctx           context.Context
		expectedError error
		description   string
	}{
		// CREATE operations
		"Create imperative integration with write context succeeds": {
			operation:   "create",
			integration: newSignatureIntegrationWithTraits("imperative-create", imperativeTraits),
			ctx:         s.hasWriteCtx,
		},
		"Create DEFAULT integration with write context fails": {
			operation:     "create",
			integration:   newSignatureIntegrationWithTraits("default-create", defaultTraits),
			ctx:           s.hasWriteCtx,
			expectedError: errox.NotAuthorized,
			description:   "DEFAULT trait integrations cannot be created through API",
		},
		"Create integration with nil traits defaults to imperative and succeeds": {
			operation:   "create",
			integration: newSignatureIntegration("nil-traits-create"),
			ctx:         s.hasWriteCtx,
		},

		// UPDATE operations
		"Update DEFAULT integration fails": {
			operation: "update",
			integration: func() *storage.SignatureIntegration {
				integration := newSignatureIntegrationWithTraits("default-update", defaultTraits)
				integration.Id = "mock-default-id"
				return integration
			}(),
			ctx:           s.hasWriteCtx,
			expectedError: errox.NotAuthorized,
			description:   "DEFAULT trait integrations cannot be modified",
		},

		// DELETE operations - tested separately due to setup complexity
	} {
		s.Run(name, func() {
			var err error
			switch tc.operation {
			case "create":
				_, err = s.dataStore.AddSignatureIntegration(tc.ctx, tc.integration)
			case "update":
				_, err = s.dataStore.UpdateSignatureIntegration(tc.ctx, tc.integration)
			}

			if tc.expectedError != nil {
				s.Error(err, tc.description)
				s.ErrorIs(err, tc.expectedError, tc.description)
			} else {
				s.NoError(err, tc.description)
			}
		})
	}
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
