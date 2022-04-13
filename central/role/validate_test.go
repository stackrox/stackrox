package role

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/stackrox/generated/storage"
	labelUtils "github.com/stackrox/stackrox/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestValidateRole(t *testing.T) {
	testCasesBad := map[string]*storage.Role{
		"name field must be set": {},
		"role must not have resourceToAccess field set": {
			Name: "name",
			ResourceToAccess: map[string]storage.Access{
				"Policy": storage.Access_READ_ACCESS,
			},
		},
		"role must reference an existing permission set": constructRole("role with no permission set", "", ""),
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
	return &storage.Role{
		Name:            name,
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
	}
}

func TestValidatePermissionSet(t *testing.T) {
	mockGoodID := permissionSetIDPrefix + "Tanis Half-Elven"
	mockBadID := "Tanis Half-Elven"
	mockName := "Hero of the Lance"
	mockGoodResource := "K8sRoleBinding"
	mockBadResource := "K8sWitchcraftAndWizardry"

	testCasesGood := map[string]*storage.PermissionSet{
		"id and name are set": {
			Id:   mockGoodID,
			Name: mockName,
		},
		"id, name, and a resource are set": {
			Id:   mockGoodID,
			Name: mockName,
			ResourceToAccess: map[string]storage.Access{
				mockGoodResource: storage.Access_READ_ACCESS,
			},
		},
	}

	testCasesBad := map[string]*storage.PermissionSet{
		"empty permissionSet": {},
		"id is missing": {
			Name: mockName,
		},
		"name is missing": {
			Id: mockGoodID,
		},
		"bad id": {
			Id:   mockBadID,
			Name: mockName,
		},
		"bad resource": {
			Id:   mockGoodID,
			Name: mockName,
			ResourceToAccess: map[string]storage.Access{
				mockBadResource: storage.Access_NO_ACCESS,
			},
		},
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

func TestValidateSimpleAccessScope(t *testing.T) {
	mockGoodID := EnsureValidAccessScopeID("42")
	mockBadID := "42"
	mockName := "Heart of Gold"
	mockDescription := "HHGTTG"
	mockGoodRules := &storage.SimpleAccessScope_Rules{
		IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
			{
				ClusterName:   "Atomic Vector Plotter",
				NamespaceName: "Advanced Tea Substitute",
			},
		},
	}
	mockBadRules := &storage.SimpleAccessScope_Rules{
		IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
			{NamespaceName: "Advanced Tea Substitute"}},
	}

	testCasesGood := map[string]*storage.SimpleAccessScope{
		"id and name are set": {
			Id:   mockGoodID,
			Name: mockName,
		},
		"id, name, and namespace label selector are set": {
			Id:    mockGoodID,
			Name:  mockName,
			Rules: mockGoodRules,
		},
		"all possible fields are set": {
			Id:          mockGoodID,
			Name:        mockName,
			Description: mockDescription,
			Rules:       mockGoodRules,
		},
		"label selector with empty rules": {
			Id:    mockGoodID,
			Name:  mockName,
			Rules: &storage.SimpleAccessScope_Rules{},
		},
		"label selector with empty requirements": {
			Id:   mockGoodID,
			Name: mockName,
			Rules: &storage.SimpleAccessScope_Rules{
				ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
					{Requirements: nil}}},
		},
	}

	testCasesBad := []struct {
		name                   string
		scope                  *storage.SimpleAccessScope
		expectedNumberOfErrors int
	}{
		{
			name:                   "empty simple access scope",
			scope:                  &storage.SimpleAccessScope{},
			expectedNumberOfErrors: 2,
		},
		{
			name:                   "id is missing",
			scope:                  &storage.SimpleAccessScope{Name: mockName},
			expectedNumberOfErrors: 1,
		}, {
			name:                   "name is missing",
			scope:                  &storage.SimpleAccessScope{Id: mockGoodID},
			expectedNumberOfErrors: 1,
		}, {
			name: "bad id",
			scope: &storage.SimpleAccessScope{
				Id:   mockBadID,
				Name: mockName,
			},
			expectedNumberOfErrors: 1,
		},
		{
			name: "bad rules",
			scope: &storage.SimpleAccessScope{
				Id:    mockGoodID,
				Name:  mockName,
				Rules: mockBadRules,
			},
			expectedNumberOfErrors: 1,
		}, {

			name: "missing cluster name in namespace rule",
			scope: &storage.SimpleAccessScope{
				Id:   mockGoodID,
				Name: mockName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{ClusterName: "Atomic Vector Plotter"}}}},
			expectedNumberOfErrors: 1,
		}, {
			name: "missing namespace name name in namespace rule",
			scope: &storage.SimpleAccessScope{
				Id:   mockGoodID,
				Name: mockName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{NamespaceName: "Advanced Tea Substitute"}}}},
			expectedNumberOfErrors: 1,
		}, {
			name: "multiple errors",
			scope: &storage.SimpleAccessScope{
				Id:   mockBadID,
				Name: mockName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{NamespaceName: "Advanced Tea Substitute"}},
					ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
						{Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{Key: "valid", Op: 42, Values: []string{"value"}},
						}}}}},
			expectedNumberOfErrors: 3,
		}, {
			name: "invalid selectors",
			scope: &storage.SimpleAccessScope{
				Id:   mockGoodID,
				Name: mockName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
						{Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{Key: "valid", Op: storage.SetBasedLabelSelector_UNKNOWN, Values: []string{"values"}},
							{Key: "valid", Op: storage.SetBasedLabelSelector_NOT_EXISTS},
						}}},
					ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
						{Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{Key: "", Op: storage.SetBasedLabelSelector_EXISTS, Values: nil},
							{Key: "valid", Op: storage.SetBasedLabelSelector_IN, Values: []string{"good"}},
							{Key: "valid", Op: storage.SetBasedLabelSelector_NOT_IN, Values: nil},
						}}}}},
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

func TestValidateSimpleAccessScopeRules(t *testing.T) {
	mockClusterName := "Infinite Improbability Drive"
	mockGoodNamespace := &storage.SimpleAccessScope_Rules_Namespace{
		ClusterName:   "Atomic Vector Plotter",
		NamespaceName: "Advanced Tea Substitute",
	}
	mockBadNamespace1 := &storage.SimpleAccessScope_Rules_Namespace{
		ClusterName: "Brownian Motion Producer",
	}
	mockBadNamespace2 := &storage.SimpleAccessScope_Rules_Namespace{
		NamespaceName: "Bambleweeny 57 Submeson Brain",
	}
	mockGoodSelector := labelUtils.LabelSelector("fleet", storage.SetBasedLabelSelector_NOT_IN, []string{"vogon"})
	mockBadSelector := &storage.SetBasedLabelSelector{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{
			{Key: "valid", Op: 42, Values: []string{"value"}},
		},
	}

	testCasesGood := map[string]*storage.SimpleAccessScope_Rules{
		"valid namespace label selector": {
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockGoodNamespace,
			},
		},
		"all possible rules are set": {
			IncludedClusters: []string{
				mockClusterName,
			},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockGoodNamespace,
			},
			ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
				mockGoodSelector,
			},
			NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
				mockGoodSelector,
			},
		},
	}

	testCasesBad := map[string]*storage.SimpleAccessScope_Rules{
		"namespace with missing namespace name": {
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockBadNamespace1,
			},
		},
		"namespace with missing cluster name": {
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockBadNamespace2,
			},
		},
		"cluster label selector with empty requirements": {
			ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
				mockBadSelector,
			},
		},
		"namespace label selector with empty requirements": {
			NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
				mockBadSelector,
			},
		},
	}

	for desc, scopeRules := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScopeRules(scopeRules)
			assert.NoErrorf(t, err, "simple access scope rules: '%+v'", scopeRules)
		})
	}

	for desc, scopeRules := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScopeRules(scopeRules)
			assert.Errorf(t, err, "simple access scope rules: '%+v'", scopeRules)
		})
	}
}

func TestGenerateAccessScopeID(t *testing.T) {
	id := GenerateAccessScopeID()
	validID := EnsureValidAccessScopeID(id)
	assert.Equal(t, id, validID)
}
