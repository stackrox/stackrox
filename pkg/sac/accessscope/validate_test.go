package accessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	labelUtils "github.com/stackrox/rox/pkg/labels"
	"github.com/stretchr/testify/assert"
)

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
		"rules are nil": nil,
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
