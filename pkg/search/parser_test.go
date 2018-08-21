package search

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
	// clusters only
	query := NewQueryBuilder().AddStrings(Cluster, "cluster1", "cluster2").Query()
	expectedRequest := &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
		Scopes: []*v1.Scope{
			{
				Cluster: "cluster1",
			},
			{
				Cluster: "cluster2",
			},
		},
	}

	// Create a parser that can handle deployemnt name and policy category.
	actualRequest, err := ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// namespaces only
	query = NewQueryBuilder().AddStrings(Namespace, "namespace1", "namespace2").Query()
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
		Scopes: []*v1.Scope{
			{
				Namespace: "namespace1",
			},
			{
				Namespace: "namespace2",
			},
		},
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// labels only
	query = NewQueryBuilder().AddStrings(LabelKey, "key1").AddStrings(LabelValue, "value1", "value2").Query()
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
		Scopes: []*v1.Scope{
			{
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value1",
				},
			},
			{
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value2",
				},
			},
		},
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// clusters, namespaces, and labels
	query = NewQueryBuilder().AddStrings(Cluster, "cluster1", "cluster2").AddStrings(Namespace, "name space1", "namespace2").AddStrings(LabelKey, "key1").AddStrings(LabelValue, "value1", "value2").Query()
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
		Scopes: []*v1.Scope{
			{
				Cluster:   "cluster1",
				Namespace: "name space1",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value1",
				},
			},
			{
				Cluster:   "cluster2",
				Namespace: "name space1",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value1",
				},
			},
			{
				Cluster:   "cluster1",
				Namespace: "namespace2",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value1",
				},
			},
			{
				Cluster:   "cluster2",
				Namespace: "namespace2",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value1",
				},
			},
			{
				Cluster:   "cluster1",
				Namespace: "name space1",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value2",
				},
			},
			{
				Cluster:   "cluster2",
				Namespace: "name space1",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value2",
				},
			},
			{
				Cluster:   "cluster1",
				Namespace: "namespace2",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value2",
				},
			},
			{
				Cluster:   "cluster2",
				Namespace: "namespace2",
				Label: &v1.Scope_Label{
					Key:   "key1",
					Value: "value2",
				},
			},
		},
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)

	// Elements match because the ordering of the scopes does not matter and is an implementation detail
	assert.ElementsMatch(t, expectedRequest.GetScopes(), actualRequest.GetScopes())

	// fields without scope
	query = NewQueryBuilder().AddStrings(DeploymentName, "field1", "field12").AddStrings(Category, "field2").Query()
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			DeploymentName: {
				Values: []string{"field1", "field12"},
			},
			Category: {
				Values: []string{"field2"},
			},
		},
		Scopes: []*v1.Scope{},
	}

	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// fields with raw query
	query = NewQueryBuilder().AddStrings(DeploymentName, "field1").AddStrings(Category, "field2").AddStringQuery("rawquery").Query()
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			DeploymentName: {
				Values: []string{"field1"},
			},
			Category: {
				Values: []string{"field2"},
			},
		},
		Scopes:      []*v1.Scope{},
		StringQuery: "rawquery",
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)
}
