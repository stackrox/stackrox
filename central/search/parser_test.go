package search

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
	// clusters only
	query := "Cluster:cluster1,cluster2"
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
	actualRequest, err := ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// namespaces only
	query = "Namespace:namespace1,namespace2"
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
	query = "Label:key1=value1,key2=value2"
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
					Key:   "key2",
					Value: "value2",
				},
			},
		},
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// bad labels
	query = "Label:key1value1"
	actualRequest, err = ParseRawQuery(query)
	assert.Error(t, err)

	// clusters, namespaces, and labels
	query = "Cluster:cluster1,cluster2+Namespace:name space1,namespace2+Label:key1=value1,key2=value2"
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
				Cluster:   "cluster1",
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
				Namespace: "name space1",
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
				Namespace: "name space1",
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
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	// Elements match because the ordering of the scopes does not matter and is an implementation detail
	assert.ElementsMatch(t, expectedRequest.GetScopes(), actualRequest.GetScopes())

	// fields without scope
	query = "Deployment Name:field1,field12+Category:field2"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"deployment.name": {
				Values: []string{"field1", "field12"},
			},
			"policy.categories": {
				Values: []string{"field2"},
			},
		},
	}

	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// fields with raw query
	query = "Deployment Name:field1+Category:field2+has:rawquery"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"deployment.name": {
				Values: []string{"field1"},
			},
			"policy.categories": {
				Values: []string{"field2"},
			},
		},
		StringQuery: "rawquery",
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)
}
