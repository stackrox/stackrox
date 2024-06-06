package policyutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
)

func TestDeploymentExclusionToQuery_Nil(t *testing.T) {
	q := DeploymentExclusionToQuery(nil)
	protoassert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_NoExclusions(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{})
	protoassert.Equal(t, q, search.MatchNoneQuery())
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
	protoassert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_MalformedDeploymentExclusion(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		{
			Name:       "myExcludedScope",
			Deployment: &storage.Exclusion_Deployment{},
		},
	})
	protoassert.Equal(t, q, search.MatchNoneQuery())
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
	protoassert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "blessed-deployment").ProtoQuery())
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
	protoassert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.ClusterID, "blessed-cluster-id").ProtoQuery())
}
