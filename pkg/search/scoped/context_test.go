package scoped

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestContext(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(scopedContextTestSuite))
}

type scopedContextTestSuite struct {
	suite.Suite
}

func (s *scopedContextTestSuite) TestGetScopeAtLevel() {
	ctx := context.Background()
	ctx = Context(ctx, Scope{
		ID:    "image-1",
		Level: v1.SearchCategory_IMAGES,
	})

	ctx = Context(ctx, Scope{
		ID:    "component-1",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})

	imageScope, hasImageScope := GetScopeAtLevel(ctx, v1.SearchCategory_IMAGES)
	s.Equal(true, hasImageScope)
	s.Equal(Scope{
		ID:     "image-1",
		Level:  v1.SearchCategory_IMAGES,
		Parent: nil,
	}, imageScope)

	componentScope, hasCompScope := GetScopeAtLevel(ctx, v1.SearchCategory_IMAGE_COMPONENTS)
	s.Equal(true, hasCompScope)
	s.Equal(Scope{
		ID:    "component-1",
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		Parent: &Scope{
			ID:     "image-1",
			Level:  v1.SearchCategory_IMAGES,
			Parent: nil,
		},
	}, componentScope)

	deploymentScope, hasDepScope := GetScopeAtLevel(ctx, v1.SearchCategory_DEPLOYMENTS)
	s.Equal(false, hasDepScope)
	s.Equal(Scope{}, deploymentScope)
}

func (s *scopedContextTestSuite) TestGetAllScopes() {
	ctx := context.Background()
	clusterScope := Scope{
		ID:    "c1",
		Level: v1.SearchCategory_CLUSTERS,
	}
	nsScope := Scope{
		ID:    "n1",
		Level: v1.SearchCategory_NAMESPACES,
	}

	clusterCtx := Context(ctx, clusterScope)
	nsCtx := Context(clusterCtx, nsScope)

	scopes, hasScope := GetAllScopes(ctx)
	s.Equal(false, hasScope)
	s.Nil(scopes)

	scopes, hasScope = GetAllScopes(nsCtx)
	s.Equal(true, hasScope)
	nsScope.Parent = &clusterScope
	s.ElementsMatch([]Scope{clusterScope, nsScope}, scopes)

	deploymentScope := Scope{
		ID:    "d1",
		Level: v1.SearchCategory_DEPLOYMENTS,
	}
	depCtx := Context(nsCtx, deploymentScope)
	scopes, hasScope = GetAllScopes(depCtx)
	s.Equal(true, hasScope)
	deploymentScope.Parent = &nsScope
	s.ElementsMatch([]Scope{clusterScope, nsScope, deploymentScope}, scopes)
}
