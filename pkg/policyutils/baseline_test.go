package policyutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentWhitelistToQuery_Nil(t *testing.T) {
	q := DeploymentBaselineToQuery(nil)
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentWhitelistToQuery_NoWhitelists(t *testing.T) {
	q := DeploymentBaselineToQuery([]*storage.Whitelist{})
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentWhitelistToQuery_NoDeploymentWhitelists(t *testing.T) {
	q := DeploymentBaselineToQuery([]*storage.Whitelist{
		{
			Name: "myExcludedScope",
			Image: &storage.Whitelist_Image{
				Name: "blessed-image",
			},
		},
	})
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentWhitelistToQuery_MalformedDeploymentWhitelist(t *testing.T) {
	q := DeploymentBaselineToQuery([]*storage.Whitelist{
		{
			Name:       "myExcludedScope",
			Deployment: &storage.Whitelist_Deployment{},
		},
	})
	assert.Equal(t, q, search.MatchNoneQuery())
}

func TestDeploymentWhitelistToQuery_NamedDeploymentWhitelist(t *testing.T) {
	q := DeploymentBaselineToQuery([]*storage.Whitelist{
		{
			Name: "myExcludedScope",
			Deployment: &storage.Whitelist_Deployment{
				Name: "blessed-deployment",
			},
		},
	})
	assert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "blessed-deployment").ProtoQuery())
}

func TestDeploymentWhitelistToQuery_ScopedDeploymentWhitelist(t *testing.T) {
	q := DeploymentBaselineToQuery([]*storage.Whitelist{
		{
			Name: "myExcludedScope",
			Deployment: &storage.Whitelist_Deployment{
				Scope: &storage.Scope{
					Cluster: "blessed-cluster-id",
				},
			},
		},
	})
	assert.Equal(t, q, search.NewQueryBuilder().AddExactMatches(search.ClusterID, "blessed-cluster-id").ProtoQuery())
}
