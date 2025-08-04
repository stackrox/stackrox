package trace

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	expectedID = "id"
)

func TestContextWithClusterID(t *testing.T) {
	ctx := ContextWithClusterID(context.Background(), &fakeClusterIDGetter{expectedID})
	assertMetadata(t, ctx, expectedID)
}

func TestBackground(t *testing.T) {
	ctx := Background(&fakeClusterIDGetter{expectedID})
	assertMetadata(t, ctx, expectedID)
}

func assertMetadata(t *testing.T, ctx context.Context, expectedID string) {
	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	actualID, ok := md[logging.ClusterIDContextValue]
	require.True(t, ok)
	require.Len(t, actualID, 1)
	assert.Equal(t, expectedID, actualID[0])
}

type fakeClusterIDGetter struct {
	id string
}

func (f *fakeClusterIDGetter) GetNoWait() string {
	return f.id
}
