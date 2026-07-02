package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/indexer"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newTestIndexerService creates a test indexerService backed by a mock Clair server.
func newTestIndexerService(t *testing.T, handler http.Handler) *indexerService {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := clairclient.NewClient(srv.URL)
	require.NoError(t, err)

	idx := indexer.NewLocalIndexer(c, nil)
	return NewIndexerService(idx)
}

func TestCreateIndexReport_Success(t *testing.T) {
	const hashID = "sha256:abc123"
	const imageURL = "docker.io/library/alpine:3.18"

	// Mock Clair server that returns a successful index report
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexer/api/v1/index_report" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return a simple index report
		report := &clairclient.IndexReport{
			ManifestHash: hashID,
			State:        "IndexFinished",
			Packages: map[string]clairclient.Package{
				"1": {
					ID:      "1",
					Name:    "alpine-baselayout",
					Version: "3.4.3-r1",
					Kind:    "binary",
				},
			},
			Environments: map[string][]clairclient.Environment{
				"1": {
					{
						PackageDB: "lib/apk/db/installed",
					},
				},
			},
		}

		w.WriteHeader(http.StatusCreated) // Clair returns 201 for CreateIndexReport
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	svc := newTestIndexerService(t, handler)

	req := &v4.CreateIndexReportRequest{
		HashId: hashID,
		ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{
			ContainerImage: &v4.ContainerImageLocator{
				Url: imageURL,
			},
		},
	}

	resp, err := svc.CreateIndexReport(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, hashID, resp.GetHashId())
	assert.Equal(t, "IndexFinished", resp.GetState())
	assert.Len(t, resp.GetContents().GetPackages(), 1)
}

func TestCreateIndexReport_NilContainerImage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call Clair")
	})

	svc := newTestIndexerService(t, handler)

	req := &v4.CreateIndexReportRequest{
		HashId:          "sha256:abc123",
		ResourceLocator: nil, // nil resource locator
	}

	resp, err := svc.CreateIndexReport(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)

	// Check for InvalidArgument status
	st := status.Convert(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetIndexReport_Success(t *testing.T) {
	const hashID = "sha256:abc123"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/indexer/api/v1/index_report/"+hashID {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		report := &clairclient.IndexReport{
			ManifestHash: hashID,
			State:        "IndexFinished",
			Packages: map[string]clairclient.Package{
				"1": {
					ID:      "1",
					Name:    "test-package",
					Version: "1.0.0",
					Kind:    "binary",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	svc := newTestIndexerService(t, handler)

	req := &v4.GetIndexReportRequest{
		HashId: hashID,
	}

	resp, err := svc.GetIndexReport(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, hashID, resp.GetHashId())
	assert.Equal(t, "IndexFinished", resp.GetState())
}

func TestGetIndexReport_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "404",
			"message": "index report not found",
		})
	})

	svc := newTestIndexerService(t, handler)

	req := &v4.GetIndexReportRequest{
		HashId: "sha256:nonexistent",
	}

	resp, err := svc.GetIndexReport(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)

	// Check for NotFound status
	st := status.Convert(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestHasIndexReport_Exists(t *testing.T) {
	const hashID = "sha256:abc123"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/indexer/api/v1/index_report/"+hashID {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return minimal report to indicate it exists
		report := &clairclient.IndexReport{
			ManifestHash: hashID,
			State:        "IndexFinished",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	svc := newTestIndexerService(t, handler)

	req := &v4.HasIndexReportRequest{
		HashId: hashID,
	}

	resp, err := svc.HasIndexReport(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.GetExists())
}

func TestHasIndexReport_NotExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "404",
			"message": "index report not found",
		})
	})

	svc := newTestIndexerService(t, handler)

	req := &v4.HasIndexReportRequest{
		HashId: "sha256:nonexistent",
	}

	resp, err := svc.HasIndexReport(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.GetExists())
}
