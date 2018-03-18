package blevesearch

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

var policyObjectMap = map[string]string{
	"policy": "",
}

// AddPolicy adds the policy to the index
func (b *Indexer) AddPolicy(policy *v1.Policy) error {
	return b.policyIndex.Index(policy.GetId(), policy)
}

// DeletePolicy deletes the policy from the index
func (b *Indexer) DeletePolicy(id string) error {
	return b.policyIndex.Delete(id)
}

func scopeToPolicyQuery(scope *v1.Scope) *query.ConjunctionQuery {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		disjunction := bleve.NewDisjunctionQuery()
		disjunction.AddQuery(newMatchQuery("scope.cluster", scope.GetCluster()))
		// Match everything then negate it
		regexQuery := bleve.NewRegexpQuery(".*")
		regexQuery.FieldVal = "scope.cluster"

		q := bleve.NewBooleanQuery()
		q.AddMustNot(regexQuery)
		disjunction.AddQuery(q)

		// This equates to either matching the cluster or having no clusters
		conjunctionQuery.AddQuery(disjunction)
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("scope.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("scope.label.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(newMatchQuery("scope.label.value", scope.GetLabel().GetValue()))
	}
	return conjunctionQuery
}

// SearchPolicies takes a SearchRequest and finds any matches
func (b *Indexer) SearchPolicies(request *v1.SearchRequest) ([]string, error) {
	return runSearchRequest(request, b.policyIndex, scopeToPolicyQuery, policyObjectMap)
}
