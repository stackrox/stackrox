package policyutils

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentExclusionToQuery_Nil(t *testing.T) {
	q := DeploymentExclusionToQuery(nil)
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_NoExclusions(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{})
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_NoDeploymentExclusions(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		{
			Name: "myExcludedScope",
			Image: &storage.Exclusion_Image{
				Name: "blessed-image",
			},
		},
	})
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_MalformedDeploymentExclusion(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		{
			Name:       "myExcludedScope",
			Deployment: &storage.Exclusion_Deployment{},
		},
	})
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_NamedDeploymentExclusion(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		{
			Name: "myExcludedScope",
			Deployment: &storage.Exclusion_Deployment{
				Name: "blessed-deployment",
			},
		},
	})
	assert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "blessed-deployment").ProtoQuery())
}

func TestDeploymentExclusionToQuery_ScopedDeploymentExclusion(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		{
			Name: "myExcludedScope",
			Deployment: &storage.Exclusion_Deployment{
				Scope: &storage.Scope{
					Cluster: "blessed-cluster-id",
				},
			},
		},
	})
	assert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.ClusterID, "blessed-cluster-id").ProtoQuery())
}
