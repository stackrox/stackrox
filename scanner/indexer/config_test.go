package indexer

import (
	"context"
	"crypto/sha256"
	"io"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/quay/claircore"
	mockIndexer "github.com/quay/claircore/test/mock/indexer"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stretchr/testify/require"
)

func TestLibindexCreation(t *testing.T) {
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
  name_to_cpe_url: https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/container-name-repos-map.json
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
	_, err = newLibindex(zlog.Test(ctx, t), ic, s, nil)
	require.NoError(t, err)
}

// loadIndexerConfig parses the provided YAML data and returns the IndexerConfig.
func loadIndexerConfig(r io.Reader) (config.IndexerConfig, error) {
	cfg, err := config.Load(r)
	if err != nil {
		return config.IndexerConfig{}, err
	}
	return cfg.Indexer, nil
}

func mockIndexerStore(t *testing.T) mockIndexer.Store {
	ctrl := gomock.NewController(t)
	s := mockIndexer.NewMockStore(ctrl)
	s.EXPECT().AffectedManifests(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		[]claircore.Digest{
			digest("first digest"),
			digest("second digest"),
		},
		nil,
	).MaxTimes(40)
	s.EXPECT().RegisterScanners(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	return s
}

func digest(inp string) claircore.Digest {
	h := sha256.New()
	_, err := h.Write([]byte(inp))
	if err != nil {
		panic(err)
	}
	d, err := claircore.NewDigest("sha256", h.Sum(nil))
	if err != nil {
		panic(err)
	}
	return d
}
