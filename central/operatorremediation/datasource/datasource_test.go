package datasource

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeImageAPI struct {
	byID       map[string]*storage.Image
	scanByRef  map[string]*storage.Image
	listResult []*storage.ListImage
	lastQuery  string
}

func (f *fakeImageAPI) GetImage(_ context.Context, req *v1.GetImageRequest) (*storage.Image, error) {
	if img, ok := f.byID[req.GetId()]; ok {
		return img, nil
	}
	return nil, errors.Wrapf(errox.NotFound, "image %q", req.GetId())
}

func (f *fakeImageAPI) ScanImage(_ context.Context, req *v1.ScanImageRequest) (*storage.Image, error) {
	if img, ok := f.scanByRef[req.GetImageName()]; ok {
		return img, nil
	}
	return nil, errors.Errorf("scan failed for %q", req.GetImageName())
}

func (f *fakeImageAPI) ListImages(_ context.Context, req *v1.RawQuery) (*v1.ListImagesResponse, error) {
	f.lastQuery = req.GetQuery()
	return &v1.ListImagesResponse{Images: f.listResult}, nil
}

func TestGetCVEsByDigestNotFound(t *testing.T) {
	c := New(&fakeImageAPI{byID: map[string]*storage.Image{}})
	cves, err := c.GetCVEsByDigest(context.Background(), "sha256:missing")
	require.NoError(t, err)
	assert.Nil(t, cves, "unknown image must map to (nil, nil)")
}

func TestListDeployedAmong(t *testing.T) {
	c := New(&fakeImageAPI{listResult: []*storage.ListImage{
		{Id: "sha256:a"}, {Id: "sha256:b"},
	}})
	got, err := c.ListDeployedAmong(context.Background(), []string{"sha256:a", "sha256:b", "sha256:c"})
	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{"sha256:a": {}, "sha256:b": {}}, got)

	// Empty input short-circuits (no query).
	empty, err := c.ListDeployedAmong(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestListDeployedAmongQueryShape(t *testing.T) {
	f := &fakeImageAPI{}
	c := New(f)
	_, err := c.ListDeployedAmong(context.Background(), []string{"sha256:a"})
	require.NoError(t, err)
	// Exact-match digest AND a match-any deployment predicate (forces the deployed inner join).
	assert.Contains(t, f.lastQuery, `Image Sha:"sha256:a"`)
	assert.Contains(t, f.lastQuery, "Deployment:")
}

func TestResolveDigest(t *testing.T) {
	c := New(&fakeImageAPI{byID: map[string]*storage.Image{
		"some-uuid": {Id: "sha256:resolved"},
	}})
	// sha passthrough (no lookup).
	d, err := c.ResolveDigest(context.Background(), "sha256:direct")
	require.NoError(t, err)
	assert.Equal(t, "sha256:direct", d)
	// uuid -> resolved sha.
	d, err = c.ResolveDigest(context.Background(), "some-uuid")
	require.NoError(t, err)
	assert.Equal(t, "sha256:resolved", d)
	// unknown -> ("", nil).
	d, err = c.ResolveDigest(context.Background(), "unknown")
	require.NoError(t, err)
	assert.Equal(t, "", d)
}
