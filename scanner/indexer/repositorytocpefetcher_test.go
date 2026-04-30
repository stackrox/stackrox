package indexer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quay/claircore/test"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryToCPEFetcher_Fetch(t *testing.T) {
	sampleMapping := repositorytocpe.MappingFile{
		Data: map[string]repositorytocpe.Repo{
			"rhel-8-server": {CPEs: []string{"cpe:/o:redhat:rhel:8"}},
			"rhel-9-server": {CPEs: []string{"cpe:/o:redhat:rhel:9", "cpe:/o:redhat:rhel:9::server"}},
		},
	}

	tests := map[string]struct {
		handler          http.HandlerFunc
		ifModifiedSince  string
		wantModified     bool
		wantLastModified string
		wantDataLen      int
		wantErr          string
	}{
		"200 OK with valid JSON": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Last-Modified", "Tue, 01 Jan 2025 00:00:00 GMT")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(sampleMapping)
			},
			wantModified:     true,
			wantLastModified: "Tue, 01 Jan 2025 00:00:00 GMT",
			wantDataLen:      2,
		},
		"304 Not Modified": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotModified)
			},
			ifModifiedSince:  "Mon, 01 Jan 2024 00:00:00 GMT",
			wantModified:     false,
			wantLastModified: "Mon, 01 Jan 2024 00:00:00 GMT",
		},
		"500 Internal Server Error": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: "unexpected status code 500 from",
		},
		"malformed JSON body": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{invalid json"))
			},
			wantErr: "decoding response",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			ctx := test.Logging(t)
			fetcher := NewRepositoryToCPEFetcher(srv.Client(), srv.URL)
			result, err := fetcher.Fetch(ctx, tt.ifModifiedSince)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantModified, result.Modified)
			assert.Equal(t, tt.wantLastModified, result.LastModified)
			if tt.wantModified {
				require.NotNil(t, result.Data)
				assert.Len(t, result.Data.Data, tt.wantDataLen)
			}
		})
	}

	t.Run("If-Modified-Since header is set when provided", func(t *testing.T) {
		var receivedHeader string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeader = r.Header.Get("If-Modified-Since")
			w.WriteHeader(http.StatusNotModified)
		}))
		defer srv.Close()

		ctx := test.Logging(t)
		fetcher := NewRepositoryToCPEFetcher(srv.Client(), srv.URL)
		_, err := fetcher.Fetch(ctx, "Tue, 01 Jan 2025 00:00:00 GMT")
		require.NoError(t, err)
		assert.Equal(t, "Tue, 01 Jan 2025 00:00:00 GMT", receivedHeader)
	})

	t.Run("If-Modified-Since header is not set when empty", func(t *testing.T) {
		var hasHeader bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, hasHeader = r.Header["If-Modified-Since"]
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(sampleMapping)
		}))
		defer srv.Close()

		ctx := test.Logging(t)
		fetcher := NewRepositoryToCPEFetcher(srv.Client(), srv.URL)
		_, err := fetcher.Fetch(ctx, "")
		require.NoError(t, err)
		assert.False(t, hasHeader)
	})
}
