package index

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/central/policy/index/mappings"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

// AlertIndex provides storage functionality for alerts.
type indexerImpl struct {
	index bleve.Index
}

type policyWrapper struct {
	*v1.Policy `json:"policy"`
	Type       string `json:"type"`
}

// AddPolicy adds the policy to the index
func (b *indexerImpl) AddPolicy(policy *v1.Policy) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Policy")
	return b.index.Index(policy.GetId(), &policyWrapper{Type: v1.SearchCategory_POLICIES.String(), Policy: policy})
}

// DeletePolicy deletes the policy from the index
func (b *indexerImpl) DeletePolicy(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Policy")
	return b.index.Delete(id)
}

// SearchPolicies takes a SearchRequest and finds any matches
func (b *indexerImpl) SearchPolicies(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Policy")
	return blevesearch.RunSearchRequest(v1.SearchCategory_POLICIES.String(), request, b.index, ScopeToPolicyQuery, mappings.OptionsMap)
}

// ScopeToPolicyQuery returns a policy query for the given scope.
func ScopeToPolicyQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		disjunction := bleve.NewDisjunctionQuery()
		disjunction.AddQuery(blevesearch.NewPrefixQuery("policy.scope.cluster", scope.GetCluster()))
		// Match everything then negate it
		regexQuery := bleve.NewRegexpQuery(".*")
		regexQuery.FieldVal = "policy.scope.cluster"

		q := bleve.NewBooleanQuery()
		q.AddMustNot(regexQuery)
		disjunction.AddQuery(q)

		// This equates to either matching the cluster or having no clusters
		conjunctionQuery.AddQuery(disjunction)
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("policy.scope.namespace", scope.GetNamespace()))
	}
	if scope.GetLabel().GetKey() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("policy.scope.label.key", scope.GetLabel().GetKey()))
	}
	if scope.GetLabel().GetValue() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewPrefixQuery("policy.scope.label.value", scope.GetLabel().GetValue()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchNoneQuery()
	}
	return conjunctionQuery
}
