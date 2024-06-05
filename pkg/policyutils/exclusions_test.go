package policyutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentExclusionToQuery_Nil(t *testing.T) {
	q := DeploymentExclusionToQuery(nil)
	assert.True(t, protocompat.Equal(q, search.MatchNoneQuery()))
}

func TestDeploymentExclusionToQuery_NoExclusions(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{})
	assert.True(t, protocompat.Equal(q, search.MatchNoneQuery()))
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
	assert.True(t, protocompat.Equal(q, search.MatchNoneQuery()))
}

func TestDeploymentExclusionToQuery_MalformedDeploymentExclusion(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		{
			Name:       "myExcludedScope",
			Deployment: &storage.Exclusion_Deployment{},
		},
	})
	assert.True(t, protocompat.Equal(q, search.MatchNoneQuery()))
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
	assert.True(t, protocompat.Equal(q, search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "blessed-deployment").ProtoQuery()))
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
	assert.True(t, protocompat.Equal(q, search.NewQueryBuilder().AddExactMatches(search.ClusterID, "blessed-cluster-id").ProtoQuery()))
}
