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

	// Create a parser that can handle deployemnt name and policy category.
	parser := &QueryParser{
		OptionsMap: map[string]*v1.SearchField{
			DeploymentName: NewStringField("deployment.name"),
			Category:       NewStringField("policy.categories"),
		},
	}
	actualRequest, err := parser.ParseRawQuery(query)
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
	actualRequest, err = parser.ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// labels only
	query = "Label Key:key1+Label Value:value1,value2"
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
	actualRequest, err = parser.ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// clusters, namespaces, and labels
	query = "Cluster:cluster1,cluster2+Namespace:name space1,namespace2+Label Key:key1+Label Value:value1,value2"
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
	actualRequest, err = parser.ParseRawQuery(query)
	assert.NoError(t, err)

	// Elements match because the ordering of the scopes does not matter and is an implementation detail
	assert.ElementsMatch(t, expectedRequest.GetScopes(), actualRequest.GetScopes())

	// fields without scope
	query = "Deployment Name:field1,field12+Category:field2"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"deployment.name": {
				Field: &v1.SearchField{
					FieldPath: "deployment.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"field1", "field12"},
			},
			"policy.categories": {
				Field: &v1.SearchField{
					FieldPath: "policy.categories",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"field2"},
			},
		},
		Scopes: []*v1.Scope{},
	}

	actualRequest, err = parser.ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// fields with raw query
	query = "Deployment Name:field1+Category:field2+has:rawquery"
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"deployment.name": {
				Field: &v1.SearchField{
					FieldPath: "deployment.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"field1"},
			},
			"policy.categories": {
				Field: &v1.SearchField{
					FieldPath: "policy.categories",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"field2"},
			},
		},
		Scopes:      []*v1.Scope{},
		StringQuery: "rawquery",
	}
	actualRequest, err = parser.ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)
}
