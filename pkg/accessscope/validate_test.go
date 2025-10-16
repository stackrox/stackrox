package accessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	labelUtils "github.com/stackrox/rox/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestValidateSimpleAccessScopeRules(t *testing.T) {
	mockClusterName := "Infinite Improbability Drive"
	mockGoodNamespace := &storage.SimpleAccessScope_Rules_Namespace{}
	mockGoodNamespace.SetClusterName("Atomic Vector Plotter")
	mockGoodNamespace.SetNamespaceName("Advanced Tea Substitute")
	mockBadNamespace1 := &storage.SimpleAccessScope_Rules_Namespace{}
	mockBadNamespace1.SetClusterName("Brownian Motion Producer")
	mockBadNamespace2 := &storage.SimpleAccessScope_Rules_Namespace{}
	mockBadNamespace2.SetNamespaceName("Bambleweeny 57 Submeson Brain")
	mockGoodSelector := labelUtils.LabelSelector("fleet", storage.SetBasedLabelSelector_NOT_IN, []string{"vogon"})
	mockBadSelector := storage.SetBasedLabelSelector_builder{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{
			storage.SetBasedLabelSelector_Requirement_builder{Key: "valid", Op: 42, Values: []string{"value"}}.Build(),
		},
	}.Build()

	sr := &storage.SimpleAccessScope_Rules{}
	sr.SetIncludedNamespaces([]*storage.SimpleAccessScope_Rules_Namespace{
		mockGoodNamespace,
	})
	sr2 := &storage.SimpleAccessScope_Rules{}
	sr2.SetIncludedClusters([]string{
		mockClusterName,
	})
	sr2.SetIncludedNamespaces([]*storage.SimpleAccessScope_Rules_Namespace{
		mockGoodNamespace,
	})
	sr2.SetClusterLabelSelectors([]*storage.SetBasedLabelSelector{
		mockGoodSelector,
	})
	sr2.SetNamespaceLabelSelectors([]*storage.SetBasedLabelSelector{
		mockGoodSelector,
	})
	testCasesGood := map[string]*storage.SimpleAccessScope_Rules{
		"valid namespace label selector": sr,
		"all possible rules are set":     sr2,
	}

	testCasesBad := map[string]*storage.SimpleAccessScope_Rules{
		"rules are nil": nil,
		"namespace with missing namespace name": storage.SimpleAccessScope_Rules_builder{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockBadNamespace1,
			},
		}.Build(),
		"namespace with missing cluster name": storage.SimpleAccessScope_Rules_builder{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockBadNamespace2,
			},
		}.Build(),
		"cluster label selector with empty requirements": storage.SimpleAccessScope_Rules_builder{
			ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
				mockBadSelector,
			},
		}.Build(),
		"namespace label selector with empty requirements": storage.SimpleAccessScope_Rules_builder{
			NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
				mockBadSelector,
			},
		}.Build(),
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
