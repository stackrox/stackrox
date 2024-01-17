package indexer

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/name"
	mockindexer "github.com/quay/claircore/test/mock/indexer"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseContainerImageURL(t *testing.T) {
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

func TestNewLibindex(t *testing.T) {
	ctx := context.Background()
	s := mockIndexerStore(t)
	yamlData := `
http_listen_addr: 127.0.0.1:9443
grpc_listen_addr: 127.0.0.1:8443
indexer:
  enable: true
  database:
    conn_string: "host=/var/run/postgresql"
    password_file: ""
  get_layer_timeout: 1m
  repository_to_cpe_url: https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/repository-to-cpe.json
  name_to_repos_url: https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/container-name-repos-map.json
matcher:
  enable: true
  database:
    conn_string: "host=/var/run/postgresql"
    password_file: ""
mtls:
  certs_dir: ""
log_level: info
`
	reader := strings.NewReader(yamlData)
	ic, err := loadIndexerConfig(reader)
	require.NoError(t, err)
	indexer, err := newLibindex(zlog.Test(ctx, t), ic, s, nil)
	require.NoError(t, err)
	assert.NotNil(t, indexer.Options.ScannerConfig.Repo["rhel-repository-scanner"])
	assert.NotNil(t, indexer.Options.ScannerConfig.Package["rhel_containerscanner"])
}

// loadIndexerConfig parses the provided YAML data and returns the IndexerConfig.
func loadIndexerConfig(r io.Reader) (config.IndexerConfig, error) {
	cfg, err := config.Load(r)
	if err != nil {
		return config.IndexerConfig{}, err
	}
	return cfg.Indexer, nil
}

func mockIndexerStore(t *testing.T) mockindexer.Store {
	ctrl := gomock.NewController(t)
	s := mockindexer.NewMockStore(ctrl)
	s.EXPECT().RegisterScanners(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	return s
}
