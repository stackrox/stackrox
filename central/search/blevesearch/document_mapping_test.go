package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestImage(t *testing.T) {
	indexer, err := NewTmpIndexer()
	assert.NoError(t, err)
	image := &v1.Image{
		Name: &v1.ImageName{
			FullName: "docker.io/nginx",
		},
	}
	err = indexer.AddImage(image)
	assert.NoError(t, err)
	images, err := indexer.SearchImages(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"image.name.full_name": {
				Values: []string{
					"docker",
				},
				Field: &v1.SearchField{
					FieldPath: "image.name.full_name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, images)
}

func TestAlert(t *testing.T) {
	indexer, err := NewTmpIndexer()
	assert.NoError(t, err)

	alert := fixtures.GetAlert()
	err = indexer.AddAlert(alert)
	assert.NoError(t, err)

	alerts, err := indexer.SearchAlerts(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"policy.name": {
				Values: []string{
					"vulnerable",
				},
				Field: &v1.SearchField{
					FieldPath: "policy.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, alerts)
}

func TestPolicy(t *testing.T) {
	indexer, err := NewTmpIndexer()
	assert.NoError(t, err)

	policy := &v1.Policy{
		Id:   "policyID",
		Name: "policy",
	}

	err = indexer.AddPolicy(policy)
	assert.NoError(t, err)

	policies, err := indexer.SearchPolicies(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"policy.name": {
				Values: []string{
					"policy",
				},
				Field: &v1.SearchField{
					FieldPath: "policy.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, policies)
}

func TestDeployment(t *testing.T) {
	indexer, err := NewTmpIndexer()
	assert.NoError(t, err)

	deployment := fixtures.GetAlert().GetDeployment()
	err = indexer.AddDeployment(deployment)
	assert.NoError(t, err)

	deployments, err := indexer.SearchDeployments(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"deployment.containers.image.name.registry": {
				Values: []string{
					"docker",
				},
				Field: &v1.SearchField{
					FieldPath: "image.name.registry",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, deployments)
}
