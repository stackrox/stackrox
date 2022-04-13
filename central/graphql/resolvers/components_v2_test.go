package resolvers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	imageMocks "github.com/stackrox/stackrox/central/image/datastore/mocks"
	imageComponentEdgeMocks "github.com/stackrox/stackrox/central/imagecomponentedge/datastore/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/edges"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	ctrl := gomock.NewController(t)
	imageDS := imageMocks.NewMockDataStore(ctrl)
	imageComponentEdgeDS := imageComponentEdgeMocks.NewMockDataStore(ctrl)

	root := &Resolver{
		ImageDataStore:              imageDS,
		ImageComponentEdgeDataStore: imageComponentEdgeDS,
	}

	// No scope; no query
	componentResolver := &imageComponentResolver{
		ctx:  context.Background(),
		root: root,
	}

	loc, err := componentResolver.Location(context.Background(), RawQuery{})
	assert.NoError(t, err)
	assert.Equal(t, "", loc)

	// With image scope; no query
	componentResolver = &imageComponentResolver{
		ctx: scoped.Context(context.Background(), scoped.Scope{
			ID:    "image1",
			Level: v1.SearchCategory_IMAGES,
		}),
		root: root,
		data: &storage.ImageComponent{
			Id: "comp1",
		},
	}

	imageComponentEdgeDS.EXPECT().Get(gomock.Any(), edges.EdgeID{ParentID: "image1", ChildID: "comp1"}.ToString()).
		Return(&storage.ImageComponentEdge{Location: "loc"}, true, nil)
	loc, err = componentResolver.Location(context.Background(), RawQuery{})
	assert.NoError(t, err)
	assert.Equal(t, "loc", loc)

	// With image scope and query; Scope takes precedence
	componentResolver = &imageComponentResolver{
		ctx: scoped.Context(context.Background(), scoped.Scope{
			ID:    "image1",
			Level: v1.SearchCategory_IMAGES,
		}),
		root: root,
		data: &storage.ImageComponent{
			Id: "comp1",
		},
	}

	query := "Deployment:dep"
	imageComponentEdgeDS.EXPECT().Get(gomock.Any(), edges.EdgeID{ParentID: "image1", ChildID: "comp1"}.ToString()).
		Return(&storage.ImageComponentEdge{Location: "loc"}, true, nil)
	loc, err = componentResolver.Location(context.Background(), RawQuery{Query: &query})
	assert.NoError(t, err)
	assert.Equal(t, "loc", loc)

	// With non-image scope; no query
	componentResolver = &imageComponentResolver{
		ctx: scoped.Context(context.Background(), scoped.Scope{
			ID:    "ns1",
			Level: v1.SearchCategory_NAMESPACES,
		}),
		root: root,
		data: &storage.ImageComponent{
			Id: "comp1",
		},
	}

	loc, err = componentResolver.Location(context.Background(), RawQuery{})
	assert.NoError(t, err)
	assert.Equal(t, "", loc)

	// With non-image scope; With query
	componentResolver = &imageComponentResolver{
		ctx: scoped.Context(context.Background(), scoped.Scope{
			ID:    "ns1",
			Level: v1.SearchCategory_NAMESPACES,
		}),
		root: root,
		data: &storage.ImageComponent{
			Id: "comp1",
		},
	}

	query = "Image Sha:image1"
	imageDS.EXPECT().Search(gomock.Any(), search.NewQueryBuilder().AddStrings(search.ImageSHA, "image1").ProtoQuery()).
		Return([]search.Result{{ID: "image1"}}, nil)
	imageComponentEdgeDS.EXPECT().Get(gomock.Any(), edges.EdgeID{ParentID: "image1", ChildID: "comp1"}.ToString()).
		Return(&storage.ImageComponentEdge{Location: "loc"}, true, nil)
	loc, err = componentResolver.Location(context.Background(), RawQuery{Query: &query})
	assert.NoError(t, err)
	assert.Equal(t, "loc", loc)
}
