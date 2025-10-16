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
	ei := &storage.Exclusion_Image{}
	ei.SetName("blessed-image")
	exclusion := &storage.Exclusion{}
	exclusion.SetName("myExcludedScope")
	exclusion.SetImage(ei)
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		exclusion,
	})
	protoassert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_MalformedDeploymentExclusion(t *testing.T) {
	exclusion := &storage.Exclusion{}
	exclusion.SetName("myExcludedScope")
	exclusion.SetDeployment(&storage.Exclusion_Deployment{})
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		exclusion,
	})
	protoassert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentExclusionToQuery_NamedDeploymentExclusion(t *testing.T) {
	ed := &storage.Exclusion_Deployment{}
	ed.SetName("blessed-deployment")
	exclusion := &storage.Exclusion{}
	exclusion.SetName("myExcludedScope")
	exclusion.SetDeployment(ed)
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		exclusion,
	})
	protoassert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "blessed-deployment").ProtoQuery())
}

func TestDeploymentExclusionToQuery_ScopedDeploymentExclusion(t *testing.T) {
	q := DeploymentExclusionToQuery([]*storage.Exclusion{
		storage.Exclusion_builder{
			Name: "myExcludedScope",
			Deployment: storage.Exclusion_Deployment_builder{
				Scope: storage.Scope_builder{
					Cluster: "blessed-cluster-id",
				}.Build(),
			}.Build(),
		}.Build(),
	})
	protoassert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.ClusterID, "blessed-cluster-id").ProtoQuery())
}
