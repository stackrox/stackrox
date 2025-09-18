package indexer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/libvuln/updates"
	mockccindexer "github.com/quay/claircore/test/mock/indexer"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	mockindexer "github.com/stackrox/rox/scanner/datastore/postgres/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// mustLoadIndexerConfig parses the provided YAML data and returns the IndexerConfig.
func mustLoadIndexerConfig(t *testing.T, r io.Reader) config.IndexerConfig {
	cfg, err := config.Load(r)
	require.NoError(t, err)
	return cfg.Indexer
}

func TestNewLibindex(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockccindexer.NewMockStore(ctrl)
	store.EXPECT().
		RegisterScanners(gomock.Any(), gomock.Any()).
		Return(nil)

	cfg := `
http_listen_addr: 127.0.0.1:9443
grpc_listen_addr: 127.0.0.1:8443
indexer:
  enable: true
  database:
    conn_string: "host=/var/run/postgresql"
    password_file: ""
  get_layer_timeout: 1m
  repository_to_cpe_url: https://security.access.redhat.com/data/metrics/repository-to-cpe.json
  name_to_repos_url: https://security.access.redhat.com/data/metrics/container-name-repos-map.json
matcher:
  enable: true
  database:
    conn_string: "host=/var/run/postgresql"
    password_file: ""
mtls:
  certs_dir: ""
log_level: info
`

	ic := mustLoadIndexerConfig(t, strings.NewReader(cfg))
	indexer, err := newLibindex(zlog.Test(context.Background(), t), ic, http.DefaultClient, "", store, nil)
	require.NoError(t, err)
	assert.NotNil(t, indexer.Options.ScannerConfig.Repo["rhel-repository-scanner"])
	assert.NotNil(t, indexer.Options.ScannerConfig.Package["rhel_containerscanner"])
	assert.NotNil(t, indexer.Options.ScannerConfig.Package["java"])
}

func TestGetIndexReport(t *testing.T) {
	ctx := zlog.Test(context.Background(), t)

	ctrl := gomock.NewController(t)
	store := mockccindexer.NewMockStore(ctrl)
	store.EXPECT().
		RegisterScanners(gomock.Any(), gomock.Any()).
		Return(nil)
	metadataStore := mockindexer.NewMockIndexerMetadataStore(ctrl)

	ccIndexer, err := libindex.New(ctx, &libindex.Options{
		Store:      store,
		Locker:     updates.NewLocalLockSource(),
		FetchArena: libindex.NewRemoteFetchArena(http.DefaultClient, ""),
		Ecosystems: ecosystems(ctx),
	}, http.DefaultClient)
	require.NoError(t, err)

	vscnrs, err := versionedScanners(ctx, ccIndexer.Ecosystems)
	require.NoError(t, err)

	indexer := &localIndexer{
		libIndex:      ccIndexer,
		vscnrs:        vscnrs,
		metadataStore: metadataStore,
	}

	// Could not get manifest metadata, so error.
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(false, errors.New("error"))
	ir, exists, err := indexer.GetIndexReport(ctx, "test")
	assert.Nil(t, ir)
	assert.False(t, exists)
	assert.Error(t, err)

	// Manifest metadata does not exist, so claim it doesn't exist.
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(false, nil)
	ir, exists, err = indexer.GetIndexReport(ctx, "test")
	assert.Nil(t, ir)
	assert.False(t, exists)
	assert.NoError(t, err)

	// Could not get manifest, so error.
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		ManifestScanned(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, errors.New("error"))
	ir, exists, err = indexer.GetIndexReport(ctx, "test")
	assert.Nil(t, ir)
	assert.False(t, exists)
	assert.Error(t, err)

	// Got manifest, and it's obsolete, so claim it doesn't exist.
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		ManifestScanned(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, nil)
	ir, exists, err = indexer.GetIndexReport(ctx, "test")
	assert.Nil(t, ir)
	assert.False(t, exists)
	assert.NoError(t, err)

	// Manifest exists and fetching Index Report errors.
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		ManifestScanned(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		IndexReport(gomock.Any(), gomock.Any()).
		Return(nil, false, errors.New("error"))
	ir, exists, err = indexer.GetIndexReport(ctx, "test")
	assert.Nil(t, ir)
	assert.False(t, exists)
	assert.Error(t, err)

	// Manifest exists, but Index Report doesn't.
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		ManifestScanned(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		IndexReport(gomock.Any(), gomock.Any()).
		Return(nil, false, nil)
	ir, exists, err = indexer.GetIndexReport(ctx, "test")
	assert.Nil(t, ir)
	assert.False(t, exists)
	assert.NoError(t, err)

	// Got manifest, and it's current, so return it.
	blankReport := &claircore.IndexReport{
		Hash: claircore.MustParseDigest("sha256:" + strings.Repeat("a", 64)),
	}
	metadataStore.EXPECT().
		ManifestExists(gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		ManifestScanned(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil)
	store.EXPECT().
		IndexReport(gomock.Any(), gomock.Any()).
		Return(blankReport, true, nil)
	ir, exists, err = indexer.GetIndexReport(ctx, "test")
	assert.Equal(t, blankReport, ir)
	assert.True(t, exists)
	assert.NoError(t, err)
}

func TestParseContainerImageURL(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    name.Reference
		wantErr string
	}{
		{
			name:    "empty URL",
			arg:     "",
			wantErr: "invalid URL",
		},
		{
			name:    "no schema",
			arg:     "foobar",
			wantErr: "invalid URL",
		},
		{
			name: "with http",
			arg:  "http://example.com/image:tag",
			want: func() name.Tag {
				t, _ := name.NewTag("example.com/image:tag", name.Insecure)
				return t
			}(),
		},
		{
			name: "with https",
			arg:  "https://example.com/image:tag",
			want: func() name.Tag {
				t, _ := name.NewTag("example.com/image:tag")
				return t
			}(),
		},
		{
			name: "with digest",
			arg:  "https://example.com/image@sha256:3d44fa76c2c83ed9296e4508b436ff583397cac0f4bad85c2b4ecc193ddb5106",
			want: func() name.Digest {
				d, _ := name.NewDigest("example.com/image@sha256:3d44fa76c2c83ed9296e4508b436ff583397cac0f4bad85c2b4ecc193ddb5106")
				return d
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseContainerImageURL(tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRandomExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	oneMinute := now.Add(1 * time.Minute)
	threeMinutes := now.Add(3 * time.Minute)

	i := &localIndexer{
		deleteIntervalStart:    int64((1 * time.Minute).Seconds()),
		deleteIntervalDuration: int64((2 * time.Minute).Seconds()),
	}

	const iterations = 1000
	for range iterations {
		expiry := i.randomExpiry(now)
		assert.False(t, expiry.Before(oneMinute))
		assert.True(t, expiry.Before(threeMinutes))
	}
}

// Testing the unexported shouldUpdateExternalIndexReport function because its
// output is particularly important to get right since it determines whether
// a record will be updated on conflict.
func Test_shouldUpdateExternalIndexReport(t *testing.T) {
	tests := []struct {
		name                 string
		incomingIndexVersion string
		savedIndexerVersion  string
		want                 bool
	}{
		{
			name:                 "incoming version is newer",
			incomingIndexVersion: "4.8.3",
			savedIndexerVersion:  "4.7.5",
			want:                 true,
		},
		{
			name:                 "saved version is newer",
			incomingIndexVersion: "4.7.5",
			savedIndexerVersion:  "4.8.3",
			want:                 false,
		},
		{
			name:                 "both versions are valid and the same",
			incomingIndexVersion: "4.8.3",
			savedIndexerVersion:  "4.8.3",
			want:                 true,
		},
		{
			name:                 "saved version is v4",
			incomingIndexVersion: "4.8.3",
			savedIndexerVersion:  "v4",
			want:                 true,
		},
		{
			name:                 "incoming versions are v4",
			incomingIndexVersion: "v4",
			savedIndexerVersion:  "4.8.3",
			want:                 false,
		},
		{
			name:                 "both versions are v4",
			incomingIndexVersion: "v4",
			savedIndexerVersion:  "v4",
			want:                 true,
		},
		{
			name:                 "incoming version is considered invalid",
			incomingIndexVersion: "vX.Y.Z",
			savedIndexerVersion:  "4.8.3",
			want:                 false,
		},
		{
			name:                 "saved version is considered invalid",
			incomingIndexVersion: "4.8.3",
			savedIndexerVersion:  "vX.Y.Z",
			want:                 true,
		},
		{
			name:                 "both versions are considered invalid",
			incomingIndexVersion: "vX.Y.Z",
			savedIndexerVersion:  "vX.Y.Z",
			want:                 true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := shouldUpdateExternalIndexReport(tt.incomingIndexVersion)
			got := f(tt.savedIndexerVersion)
			assert.Equal(t, tt.want, got)
		})
	}
}
