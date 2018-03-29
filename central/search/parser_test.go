package search

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
	// clusters only
	query := "cluster:cluster1 cluster:cluster2"
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
	actualRequest, err := ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// namespaces only
	query = "namespace:namespace1 namespace:namespace2"
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
	actualRequest, err = ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// labels only
	query = "label:key1=value1 label:key1=value1"
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
					Value: "value1",
				},
			},
		},
	}
	actualRequest, err = ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// bad labels
	query = "label:key1value1"
	actualRequest, err = ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.Error(t, err)

	// clusters, namespaces, and labels
	query = "cluster:cluster1 cluster:cluster2 namespace:namespace1 namespace:namespace2 label:key1=value1 label:key2=value2"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
		Scopes: []*v1.Scope{
			{
				Cluster:   "cluster1",
				Namespace: "namespace1",
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
				Cluster:   "cluster1",
				Namespace: "namespace1",
				Label: &v1.Scope_Label{
					Key:   "key2",
					Value: "value2",
				},
			},
			{
				Cluster:   "cluster1",
				Namespace: "namespace2",
				Label: &v1.Scope_Label{
					Key:   "key2",
					Value: "value2",
				},
			},
			{
				Cluster:   "cluster2",
				Namespace: "namespace1",
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
				Cluster:   "cluster2",
				Namespace: "namespace1",
				Label: &v1.Scope_Label{
					Key:   "key2",
					Value: "value2",
				},
			},
			{
				Cluster:   "cluster2",
				Namespace: "namespace2",
				Label: &v1.Scope_Label{
					Key:   "key2",
					Value: "value2",
				},
			},
		},
	}
	actualRequest, err = ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.NoError(t, err)
	// Elements match because the ordering of the scopes does not matter and is an implementation detail
	assert.ElementsMatch(t, expectedRequest.GetScopes(), actualRequest.GetScopes())

	// fields without scope
	query = "field1:field1 field1:field12 field2:field2"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"field1": {
				Values: []string{"field1", "field12"},
			},
			"field2": {
				Values: []string{"field2"},
			},
		},
	}

	actualRequest, err = ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// fields with raw query
	query = "field1:field1 field2:field2 rawquery"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"field1": {
				Values: []string{"field1"},
			},
			"field2": {
				Values: []string{"field2"},
			},
		},
		StringQuery: "rawquery",
	}
	actualRequest, err = ParseRawQuery(&v1.RawSearchRequest{Query: query})
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)
}
