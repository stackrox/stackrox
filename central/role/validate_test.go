package role

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateRole(t *testing.T) {
	role := &storage.Role{}
	role.SetName("name")
	role.SetResourceToAccess(map[string]storage.Access{
		"Policy": storage.Access_READ_ACCESS,
	})
	testCasesBad := map[string]*storage.Role{
		"name field must be set":                         {},
		"role must not have resourceToAccess field set":  role,
		"role must reference an existing permission set": constructRole("role with no permission set", "", GenerateAccessScopeID()),
		"empty access scope reference is not allowed":    constructRole("role with no access scope", GeneratePermissionSetID(), ""),
	}

	testCasesGood := map[string]*storage.Role{
		"valid name, permissionSetId and accessScopeId": constructRole("new valid role", GeneratePermissionSetID(), GenerateAccessScopeID()),
	}

	for desc, role := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateRole(role)
			assert.NoErrorf(t, err, "role: '%+v'", role)
		})
	}

	for desc, role := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidateRole(role)
			assert.Errorf(t, err, "role: '%+v'", role)
		})
	}
}

func constructRole(name, permissionSetID, accessScopeID string) *storage.Role {
	role := &storage.Role{}
	role.SetName(name)
	role.SetPermissionSetId(permissionSetID)
	role.SetAccessScopeId(accessScopeID)
	return role
}

func TestValidatePermissionSet(t *testing.T) {
	mockGoodID := uuid.NewDummy().String()
	mockBadID := "Tanis Half-Elven"
	mockName := "Hero of the Lance"
	mockGoodResource := "K8sRoleBinding"
	mockBadResource := "K8sWitchcraftAndWizardry"

	ps := &storage.PermissionSet{}
	ps.SetId(mockGoodID)
	ps.SetName(mockName)
	ps2 := &storage.PermissionSet{}
	ps2.SetId(mockGoodID)
	ps2.SetName(mockName)
	ps2.SetResourceToAccess(map[string]storage.Access{
		mockGoodResource: storage.Access_READ_ACCESS,
	})
	testCasesGood := map[string]*storage.PermissionSet{
		"id and name are set":              ps,
		"id, name, and a resource are set": ps2,
	}

	testCasesBad := map[string]*storage.PermissionSet{
		"empty permissionSet": {},
		"id is missing": storage.PermissionSet_builder{
			Name: mockName,
		}.Build(),
		"name is missing": storage.PermissionSet_builder{
			Id: mockGoodID,
		}.Build(),
		"bad id": storage.PermissionSet_builder{
			Id:   mockBadID,
			Name: mockName,
		}.Build(),
		"bad resource": storage.PermissionSet_builder{
			Id:   mockGoodID,
			Name: mockName,
			ResourceToAccess: map[string]storage.Access{
				mockBadResource: storage.Access_NO_ACCESS,
			},
		}.Build(),
	}

	for desc, permissionSet := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidatePermissionSet(permissionSet)
			assert.NoErrorf(t, err, "permission set: '%+v'", permissionSet)
		})
	}

	for desc, permissionSet := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidatePermissionSet(permissionSet)
			assert.Errorf(t, err, "permission set: '%+v'", permissionSet)
		})
	}
}

func TestGeneratePermissionSetID(t *testing.T) {
	generatedID := GeneratePermissionSetID()
	_, err := uuid.FromString(generatedID)
	assert.NoError(t, err)
}

func TestEnsureValidPermissionSetIDPostgres(t *testing.T) {
	validID := GeneratePermissionSetID()
	checkedValidID := EnsureValidPermissionSetID(validID)
	assert.Equal(t, validID, checkedValidID)

	// Test that an invalid ID triggers the generation of a valid UUID.
	invalidID := "abcdefgh-ijkl-mnop-qrst-uvwxyz012345"
	checkedInvalidID := EnsureValidPermissionSetID(invalidID)
	assert.NotEqual(t, invalidID, checkedInvalidID)
	_, err := uuid.FromString(checkedInvalidID)
	assert.NoError(t, err)
}

func TestValidateSimpleAccessScope(t *testing.T) {
	mockGoodID := uuid.NewDummy().String()
	mockBadID := "42"
	emptyID := ""
	mockName := "Heart of Gold"
	mockDescription := "HHGTTG"
	srn := &storage.SimpleAccessScope_Rules_Namespace{}
	srn.SetClusterName("Atomic Vector Plotter")
	srn.SetNamespaceName("Advanced Tea Substitute")
	mockGoodRules := &storage.SimpleAccessScope_Rules{}
	mockGoodRules.SetIncludedNamespaces([]*storage.SimpleAccessScope_Rules_Namespace{
		srn,
	})
	srn2 := &storage.SimpleAccessScope_Rules_Namespace{}
	srn2.SetNamespaceName("Advanced Tea Substitute")
	mockBadRules := &storage.SimpleAccessScope_Rules{}
	mockBadRules.SetIncludedNamespaces([]*storage.SimpleAccessScope_Rules_Namespace{
		srn2})

	testCasesGood := map[string]*storage.SimpleAccessScope{
		"id, name, and namespace label selector are set": storage.SimpleAccessScope_builder{
			Id:    mockGoodID,
			Name:  mockName,
			Rules: mockGoodRules,
		}.Build(),
		"all possible fields are set": storage.SimpleAccessScope_builder{
			Id:          mockGoodID,
			Name:        mockName,
			Description: mockDescription,
			Rules:       mockGoodRules,
		}.Build(),
		"label selector with empty rules": storage.SimpleAccessScope_builder{
			Id:    mockGoodID,
			Name:  mockName,
			Rules: &storage.SimpleAccessScope_Rules{},
		}.Build(),
		"label selector with empty requirements": storage.SimpleAccessScope_builder{
			Id:   mockGoodID,
			Name: mockName,
			Rules: storage.SimpleAccessScope_Rules_builder{
				ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
					storage.SetBasedLabelSelector_builder{Requirements: nil}.Build()}}.Build(),
		}.Build(),
	}

	testCasesBad := []struct {
		name                   string
		scope                  *storage.SimpleAccessScope
		expectedNumberOfErrors int
	}{
		{
			name:                   "empty simple access scope",
			scope:                  &storage.SimpleAccessScope{},
			expectedNumberOfErrors: 3,
		},
		{
			name:                   "id is missing",
			scope:                  storage.SimpleAccessScope_builder{Name: mockName, Rules: &storage.SimpleAccessScope_Rules{}}.Build(),
			expectedNumberOfErrors: 1,
		},
		{
			name:                   "empty id",
			scope:                  storage.SimpleAccessScope_builder{Id: emptyID, Name: mockName, Rules: &storage.SimpleAccessScope_Rules{}}.Build(),
			expectedNumberOfErrors: 1,
		}, {
			name:                   "name is missing",
			scope:                  storage.SimpleAccessScope_builder{Id: mockGoodID, Rules: &storage.SimpleAccessScope_Rules{}}.Build(),
			expectedNumberOfErrors: 1,
		}, {
			name: "bad id",
			scope: storage.SimpleAccessScope_builder{
				Id:    mockBadID,
				Name:  mockName,
				Rules: &storage.SimpleAccessScope_Rules{},
			}.Build(),
			expectedNumberOfErrors: 1,
		},
		{
			name: "bad rules",
			scope: storage.SimpleAccessScope_builder{
				Id:    mockGoodID,
				Name:  mockName,
				Rules: mockBadRules,
			}.Build(),
			expectedNumberOfErrors: 1,
		}, {

			name: "missing cluster name in namespace rule",
			scope: storage.SimpleAccessScope_builder{
				Id:   mockGoodID,
				Name: mockName,
				Rules: storage.SimpleAccessScope_Rules_builder{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						storage.SimpleAccessScope_Rules_Namespace_builder{ClusterName: "Atomic Vector Plotter"}.Build()}}.Build()}.Build(),
			expectedNumberOfErrors: 1,
		}, {
			name: "missing namespace name name in namespace rule",
			scope: storage.SimpleAccessScope_builder{
				Id:   mockGoodID,
				Name: mockName,
				Rules: storage.SimpleAccessScope_Rules_builder{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						storage.SimpleAccessScope_Rules_Namespace_builder{NamespaceName: "Advanced Tea Substitute"}.Build()}}.Build()}.Build(),
			expectedNumberOfErrors: 1,
		}, {
			name: "multiple errors",
			scope: storage.SimpleAccessScope_builder{
				Id:   mockBadID,
				Name: mockName,
				Rules: storage.SimpleAccessScope_Rules_builder{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						storage.SimpleAccessScope_Rules_Namespace_builder{NamespaceName: "Advanced Tea Substitute"}.Build()},
					ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
						storage.SetBasedLabelSelector_builder{Requirements: []*storage.SetBasedLabelSelector_Requirement{
							storage.SetBasedLabelSelector_Requirement_builder{Key: "valid", Op: 42, Values: []string{"value"}}.Build(),
						}}.Build()}}.Build()}.Build(),
			expectedNumberOfErrors: 3,
		}, {
			name: "invalid selectors",
			scope: storage.SimpleAccessScope_builder{
				Id:   mockGoodID,
				Name: mockName,
				Rules: storage.SimpleAccessScope_Rules_builder{
					NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
						storage.SetBasedLabelSelector_builder{Requirements: []*storage.SetBasedLabelSelector_Requirement{
							storage.SetBasedLabelSelector_Requirement_builder{Key: "valid", Op: storage.SetBasedLabelSelector_UNKNOWN, Values: []string{"values"}}.Build(),
							storage.SetBasedLabelSelector_Requirement_builder{Key: "valid", Op: storage.SetBasedLabelSelector_NOT_EXISTS}.Build(),
						}}.Build()},
					ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
						storage.SetBasedLabelSelector_builder{Requirements: []*storage.SetBasedLabelSelector_Requirement{
							storage.SetBasedLabelSelector_Requirement_builder{Key: "", Op: storage.SetBasedLabelSelector_EXISTS, Values: nil}.Build(),
							storage.SetBasedLabelSelector_Requirement_builder{Key: "valid", Op: storage.SetBasedLabelSelector_IN, Values: []string{"good"}}.Build(),
							storage.SetBasedLabelSelector_Requirement_builder{Key: "valid", Op: storage.SetBasedLabelSelector_NOT_IN, Values: nil}.Build(),
						}}.Build()}}.Build()}.Build(),
			expectedNumberOfErrors: 3,
		},
	}

	for desc, scope := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScope(scope)
			assert.NoErrorf(t, err, "simple access scope: '%+v'", scope)
		})
	}

	for _, tc := range testCasesBad {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSimpleAccessScope(tc.scope)
			var target *multierror.Error
			if errors.As(err, &target) {
				assert.Equal(t, tc.expectedNumberOfErrors, target.Len())
			} else {
				assert.Zero(t, tc.expectedNumberOfErrors)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateAccessScopeID(t *testing.T) {
	generatedID := GenerateAccessScopeID()
	validID := EnsureValidAccessScopeID(generatedID)
	assert.Equal(t, generatedID, validID)
	_, err := uuid.FromString(generatedID)
	assert.NoError(t, err)
}

func TestEnsureValidAccessScopeID(t *testing.T) {
	validID := GenerateAccessScopeID()
	checkedValidID := EnsureValidAccessScopeID(validID)
	assert.Equal(t, validID, checkedValidID)

	// Test that an invalid ID triggers the generation of a valid UUID.
	invalidID := "abcdefgh-ijkl-mnop-qrst-uvwxyz012345"
	checkedInvalidID := EnsureValidAccessScopeID(invalidID)
	assert.NotEqual(t, invalidID, checkedInvalidID)
	_, err := uuid.FromString(checkedInvalidID)
	assert.NoError(t, err)
}
