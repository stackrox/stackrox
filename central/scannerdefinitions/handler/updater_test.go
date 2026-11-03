package handler

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	v2UUID            = "e799c68a-671f-44db-9682-f24248cd0ffe"
	v2DiffURI         = "/" + v2UUID + "/diff.zip"
	v2ManifestContent = `{"since":"yesterday","until":"today"}`

	v4Dev         = "dev"
	v4V1          = "v1"
	v4VulnBaseURI = "/v4/vulnerability-bundles"
	v4DevURI      = v4VulnBaseURI + "/" + v4Dev + "/vulnerabilities.zip"
	v4V1URI       = v4VulnBaseURI + "/" + v4V1 + "/vulnerabilities.zip"
	v4MappingURI  = "/v4/redhat-repository-mappings/mapping.zip"
	name2repos    = `{
	"data": {
		"sample": ["sample"]
	}
}`
	repo2cpe = `{
	"data": {
		"sample": {
			"cpes": ["cpe"],
			"repo_relative_urls": ["url"]
		}
	}
}`
)

var (
	april2025 = time.Date(2025, time.April, 30, 0, 0, 0, 0, time.UTC)
	nov23     = time.Date(2019, time.November, 23, 0, 0, 0, 0, time.UTC)

	unixEpochTime = time.Unix(0, 0)
)

// isZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
//
// This is lifted from https://github.com/golang/go/blob/go1.24.2/src/net/http/fs.go#L612.
func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

// modifiedSince is based on https://github.com/golang/go/blob/go1.24.2/src/net/http/fs.go#L557.
func modifiedSince(r *http.Request, modtime time.Time) bool {
	ims := r.Header.Get("If-Modified-Since")
	if ims == "" || isZeroTime(modtime) {
		return true
	}
	t, err := http.ParseTime(ims)
	if err != nil {
		return true
	}
	// The Last-Modified header truncates sub-second precision so
	// the modtime needs to be truncated too.
	modtime = modtime.Truncate(time.Second)
	if ret := modtime.Compare(t); ret <= 0 {
		return false
	}
	return true
}

// startMockDefinitionsStackRoxIO mocks definitions.stackrox.io.
func startMockDefinitionsStackRoxIO(t *testing.T) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		var bundle *bytes.Buffer
		switch r.RequestURI {
		case v2DiffURI:
			bundle = mustCreateV2Bundle(t)
		case v4DevURI, v4V1URI:
			bundle = newZipBuilder(t).
				addFile("Scanner V4", "Scanner V4", []byte("")).
				buildBuffer()
		case v4MappingURI:
			bundle = mustCreateV4MappingBundle(t)
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Last-Modified", april2025.Format(http.TimeFormat))

		if !modifiedSince(r, april2025) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		_, err := io.Copy(w, bundle)
		require.NoError(t, err)
	}))
	// Replace definitions.stackrox.io with the mock server.
	originalURL := scannerUpdateBaseURL
	scannerUpdateBaseURL, _ = url.Parse(srv.URL)
	t.Setenv(env.ScannerV4VulnerabilityBundleBaseURL.EnvVar(), srv.URL+v4VulnBaseURI)
	t.Cleanup(func() {
		srv.Close()
		// Revert URL change.
		scannerUpdateBaseURL = originalURL
	})
	return srv.URL
}

// startBrokenMockDefinitionsStackRoxIO mocks definitions.stackrox.io but always
// returns an error.
func startBrokenMockDefinitionsStackRoxIO(t *testing.T) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	// Replace definitions.stackrox.io with the mock server.
	originalURL := scannerUpdateBaseURL
	scannerUpdateBaseURL, _ = url.Parse(srv.URL)
	t.Setenv(env.ScannerV4VulnerabilityBundleBaseURL.EnvVar(), srv.URL+v4VulnBaseURI)
	t.Cleanup(func() {
		srv.Close()
		// Revert URL change.
		scannerUpdateBaseURL = originalURL
	})
	return srv.URL
}

// mustCreateV2Bundle creates a ZIP file mimicking a Scanner v2 diff.zip.
// This ZIP contains at least one file called manifest.json.
func mustCreateV2Bundle(t *testing.T) *bytes.Buffer {
	bundle := newZipBuilder(t).
		addFile("manifest.json", "Scanner v2 manifest", []byte(v2ManifestContent)).
		buildBuffer()
	return bundle
}

// mustCreateV4MappingBundle creates a ZIP file mimicking a Scanner V4 mapping.zip.
func mustCreateV4MappingBundle(t *testing.T) *bytes.Buffer {
	bundle := newZipBuilder(t).
		addFile("repomapping/container-name-repos-map.json", "name2repos", []byte(name2repos)).
		addFile("repomapping/repository-to-cpe.json", "repo2cpe", []byte(repo2cpe)).
		buildBuffer()
	return bundle
}

type zipBuilder struct {
	t   *testing.T
	buf *bytes.Buffer
	zw  *zip.Writer
}

func newZipBuilder(t *testing.T) *zipBuilder {
	var buf bytes.Buffer
	return &zipBuilder{
		t:   t,
		buf: &buf,
		zw:  zip.NewWriter(&buf),
	}
}

func (b *zipBuilder) addFile(name, comment string, content []byte) *zipBuilder {
	require.NotNil(b.t, b.buf)
	file, err := b.zw.CreateHeader(&zip.FileHeader{
		Name:               name,
		Comment:            comment,
		UncompressedSize64: uint64(len(content)),
	})
	require.NoError(b.t, err)
	_, err = file.Write(content)
	require.NoError(b.t, err)
	return b
}

func (b *zipBuilder) buildBuffer() *bytes.Buffer {
	require.NoError(b.t, b.zw.Close())
	buf := b.buf
	*b = zipBuilder{}
	return buf
}

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

func mustGetModTime(t *testing.T, path string) time.Time {
	fi, err := os.Stat(path)
	require.NoError(t, err)
	return fi.ModTime().UTC()
}

func mustSetModTime(t *testing.T, path string, modTime time.Time) {
	require.NoError(t, os.Chtimes(path, time.Now(), modTime))
}

func TestUpdate(t *testing.T) {
	url := startMockDefinitionsStackRoxIO(t)

	filePath := filepath.Join(t.TempDir(), "dump.zip")
	u := newUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, url+v2DiffURI, 1*time.Hour)
	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	lastUpdatedTime := time.Now()
	mustSetModTime(t, filePath, lastUpdatedTime)
	// Should not fetch since we are tracking a file newer than what's on the server.
	require.NoError(t, u.doUpdate())
	assert.Equal(t, lastUpdatedTime.UTC(), mustGetModTime(t, filePath))
	assertOnFileExistence(t, filePath, true)

	// Should definitely fetch since we now have a file much older than what's on the server.
	mustSetModTime(t, filePath, nov23)
	require.NoError(t, u.doUpdate())
	assert.True(t, lastUpdatedTime.UTC().After(mustGetModTime(t, filePath)))
	assert.True(t, mustGetModTime(t, filePath).After(nov23.UTC()))
	assertOnFileExistence(t, filePath, true)
}

func TestMappingUpdate(t *testing.T) {
	url := startMockDefinitionsStackRoxIO(t)

	filePath := filepath.Join(t.TempDir(), "test.zip")
	u := newUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, url+v4MappingURI, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	n, err := countFilesInZip(filePath)
	if err != nil {
		t.Fatalf("Failed to count files in zip: %v", err)
	}
	assert.Equal(t, len(v4FileMapping), n)
}

func TestV4VulnUpdate(t *testing.T) {
	url := startMockDefinitionsStackRoxIO(t)

	filePath := filepath.Join(t.TempDir(), "test.zip")
	u := newUpdater(file.New(filePath), &http.Client{Timeout: 1 * time.Minute}, url+v4DevURI, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)
}

func TestBuildUpdateURL(t *testing.T) {
	originalURL := scannerUpdateBaseURL
	scannerUpdateBaseURL = mustParseURL(t, "https://definitions.example/base")
	t.Cleanup(func() {
		scannerUpdateBaseURL = originalURL
	})

	tests := map[string]struct {
		updaterType updaterType
		urlPath     string
		vulnBaseURL string
		expectedURL string
		expectError bool
	}{
		"default vulnerability base with dev stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "dev/vulnerabilities.zip",
			expectedURL: "https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip",
		},
		"default vulnerability base with release stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "v4/vulnerabilities.zip",
			expectedURL: "https://definitions.stackrox.io/v4/vulnerability-bundles/v4/vulnerabilities.zip",
		},
		"custom vulnerability base with dev stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "dev/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectedURL: "https://candidate.example/test-runs/123/1/dev/vulnerabilities.zip",
		},
		"custom vulnerability base with release stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "v4/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1/",
			expectedURL: "https://candidate.example/test-runs/123/1/v4/vulnerabilities.zip",
		},
		"custom vulnerability base with release candidate stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "v4-rc/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectedURL: "https://candidate.example/test-runs/123/1/v4-rc/vulnerabilities.zip",
		},
		"custom vulnerability base does not change mappings": {
			updaterType: mappingUpdaterType,
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectedURL: "https://definitions.example/base/v4/redhat-repository-mappings/mapping.zip",
		},
		"custom vulnerability base does not change Scanner V2": {
			updaterType: v2UpdaterType,
			urlPath:     v2UUID,
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectedURL: "https://definitions.example/base/" + v2UUID + "/diff.zip",
		},
		"invalid vulnerability base": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "dev/vulnerabilities.zip",
			vulnBaseURL: "relative/path",
			expectError: true,
		},
		"path traversal vulnerability stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "../../../admin/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectError: true,
		},
		"nested vulnerability stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "v4/rc/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectError: true,
		},
		"invalid percent escape in vulnerability stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "%zz/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectError: true,
		},
		"encoded separator in vulnerability stream": {
			updaterType: vulnerabilityUpdaterType,
			urlPath:     "%2Fadmin/vulnerabilities.zip",
			vulnBaseURL: "https://candidate.example/test-runs/123/1",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.ScannerV4VulnerabilityBundleBaseURL.EnvVar(), tt.vulnBaseURL)
			updateURL, err := buildUpdateURL(tt.updaterType, tt.urlPath)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURL, updateURL.String())
		})
	}
}

func TestValidateVulnerabilityStream(t *testing.T) {
	tests := map[string]struct {
		stream      string
		expectError bool
	}{
		"development stream": {
			stream: "dev",
		},
		"versioned stream": {
			stream: "v4",
		},
		"release candidate stream": {
			stream: "v4-rc",
		},
		"legacy numeric stream": {
			stream: "4.5",
		},
		"empty stream": {
			expectError: true,
		},
		"dot stream": {
			stream:      ".",
			expectError: true,
		},
		"dot dot stream": {
			stream:      "..",
			expectError: true,
		},
		"embedded dot dot": {
			stream:      "v4..rc",
			expectError: true,
		},
		"forward slash": {
			stream:      "v4/rc",
			expectError: true,
		},
		"backslash": {
			stream:      `v4\rc`,
			expectError: true,
		},
		"percent escape": {
			stream:      "%2Fadmin",
			expectError: true,
		},
		"invalid percent escape": {
			stream:      "%zz",
			expectError: true,
		},
		"space": {
			stream:      "v4 rc",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateVulnerabilityStream(tt.stream)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRejectInvalidVulnerabilityStreamsBeforeUpdaterLookup(t *testing.T) {
	h := &httpHandler{
		updaters: map[string]*requestedUpdater{
			"vulnerabilityUpdaterType-v4-rc-vulnerabilities.zip": {},
		},
	}

	_, err := h.getUpdater(vulnerabilityUpdaterType, "v4/rc/vulnerabilities.zip")
	require.Error(t, err)
}

func TestServeHTTPRejectsInvalidVulnerabilityStreams(t *testing.T) {
	tests := map[string]string{
		"dot dot":                  "version=..",
		"path traversal":           "version=..%2F..%2F..%2Fadmin",
		"encoded slash collision":  "version=v4%2Frc",
		"invalid percent escape":   "version=%25zz",
		"double encoded separator": "version=%252Fadmin",
	}

	for name, rawQuery := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://central.stackrox.svc/scannerdefinitions?"+rawQuery, nil)
			w := httptest.NewRecorder()
			(&httpHandler{}).ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	return u
}

// countFilesInZip counts the number of files inside a zip archive.
func countFilesInZip(zipFilePath string) (int, error) {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Errorf("Error closing zip reader: %v", err)
		}
	}()

	count := 0
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			count++
		}
	}

	return count, nil
}
