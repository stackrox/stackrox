//go:build sql_integration

package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/suite"
)

const (
	content1 = "Hello, world!"
	content2 = "Papaya"

	v2UUID            = "e799c68a-671f-44db-9682-f24248cd0ffe"
	v2ManifestContent = `{"since":"yesterday","until":"today"}`
	v4Dev             = "dev"
	v4V1              = "v1"

	name2repos = `{
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

var april2025 = time.Date(2025, time.April, 30, 0, 0, 0, 0, time.UTC)

type zipBuilder struct {
	buf *bytes.Buffer
	zw  *zip.Writer
}

func newZipBuilder() *zipBuilder {
	var buf bytes.Buffer
	return &zipBuilder{
		buf: &buf,
		zw:  zip.NewWriter(&buf),
	}
}

func (b *zipBuilder) addFile(s *handlerTestSuite, name, comment string, content []byte) *zipBuilder {
	s.Require().NotNil(b.buf)
	file, err := b.zw.CreateHeader(&zip.FileHeader{
		Name:               name,
		Comment:            comment,
		UncompressedSize64: uint64(len(content)),
	})
	s.Require().NoError(err)
	_, err = file.Write(content)
	s.Require().NoError(err)
	return b
}

func (b *zipBuilder) buildBuffer(s *handlerTestSuite) *bytes.Buffer {
	s.Require().NoError(b.zw.Close())
	buf := b.buf
	*b = zipBuilder{}
	return buf
}

type handlerTestSuite struct {
	suite.Suite
	ctx       context.Context
	datastore datastore.Datastore
	testDB    *pgtest.TestPostgres
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

func (s *handlerTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	blobStore := store.New(s.testDB.DB)
	s.datastore = datastore.NewDatastore(blobStore, nil)
	s.T().Setenv("TMPDIR", s.T().TempDir())
}

func (s *handlerTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

// mustCreateV2Bundle creates a ZIP file mimicking a Scanner v2 diff.zip.
// This ZIP contains at least one file called manifest.json.
func (s *handlerTestSuite) mustCreateV2Bundle() *bytes.Buffer {
	bundle := newZipBuilder().
		addFile(s, "manifest.json", "Scanner v2 manifest", []byte(v2ManifestContent)).
		buildBuffer(s)
	return bundle
}

// mustCreateV4MappingBundle creates a ZIP file mimicking a Scanner V4 mapping.zip.
func (s *handlerTestSuite) mustCreateV4MappingBundle() *bytes.Buffer {
	bundle := newZipBuilder().
		addFile(s, "repomapping/container-name-repos-map.json", "name2repos", []byte(name2repos)).
		addFile(s, "repomapping/repository-to-cpe.json", "repo2cpe", []byte(repo2cpe)).
		buildBuffer(s)
	return bundle
}

// startMockDefinitionsStackRoxIO mocks definitions.stackrox.io.
func (s *handlerTestSuite) startMockDefinitionsStackRoxIO() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		var bundle *bytes.Buffer
		switch r.RequestURI {
		case fmt.Sprintf("/%s/diff.zip", v2UUID):
			bundle = s.mustCreateV2Bundle()
		case fmt.Sprintf("/v4/vulnerability-bundles/%s/vulnerabilities.zip", v4Dev),
			fmt.Sprintf("/v4/vulnerability-bundles/%s/vulnerabilities.zip", v4V1):
			bundle = newZipBuilder().
				addFile(s, "Scanner V4", "Scanner V4", []byte("")).
				buildBuffer(s)
		case "/v4/redhat-repository-mappings/mapping.zip":
			bundle = s.mustCreateV4MappingBundle()
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Last-Modified", april2025.Format(http.TimeFormat))
		_, err := io.Copy(w, bundle)
		s.Require().NoError(err)
	}))
	// Replace definitions.stackrox.io with the mock server.
	originalURL := scannerUpdateBaseURL
	scannerUpdateBaseURL, _ = url.Parse(srv.URL)
	s.T().Cleanup(func() {
		srv.Close()
		// Revert URL change.
		scannerUpdateBaseURL = originalURL
	})
}

func (s *handlerTestSuite) postRequestV2() *http.Request {
	v2Bundle := s.mustCreateV2Bundle()
	// This mimics a real offline-bundle, which is a ZIP of ZIPs.
	// This one solely contains scanner-defs.zip, as that's all that's needed
	// for StackRox Scanner.
	bundle := newZipBuilder().
		addFile(s, "scanner-defs.zip", "Scanner v2 content", v2Bundle.Bytes()).
		buildBuffer(s)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", bundle)
	s.Require().NoError(err)

	return req
}

func (s *handlerTestSuite) postRequestV4(body io.Reader) *http.Request {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", body)
	s.Require().NoError(err)
	return req
}

func (s *handlerTestSuite) mustWriteV2Blob(content string, modTime time.Time) {
	modifiedTime, err := protocompat.ConvertTimeToTimestampOrError(modTime)
	s.Require().NoError(err)
	blob := &storage.Blob{
		Name:         offlineScannerV2DefsBlobName,
		Length:       int64(len(content)),
		ModifiedTime: modifiedTime,
		LastUpdated:  protocompat.TimestampNow(),
	}
	s.Require().NoError(s.datastore.Upsert(s.ctx, blob, bytes.NewBuffer([]byte(content))))
}

func (s *handlerTestSuite) getRequestUUID() *http.Request {
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=" + v2UUID
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	s.Require().NoError(err)

	return req
}

func (s *handlerTestSuite) getRequestFile(file string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?file=%s", file)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	s.Require().NoError(err)

	return req
}

func (s *handlerTestSuite) getRequestVersion(v string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?version=%s", v)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	s.Require().NoError(err)

	return req
}

func (s *handlerTestSuite) getRequestUUIDAndFile(file string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?uuid=%s&file=%s", v2UUID, file)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	s.Require().NoError(err)

	return req
}

func (s *handlerTestSuite) getRequestBadUUID() *http.Request {
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=fail"
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	s.Require().NoError(err)

	return req
}

func (s *handlerTestSuite) upsertBlob(zipF *zip.File, blobName string) error {
	r, err := zipF.Open()
	s.Require().NoError(err)
	defer utils.IgnoreError(r.Close)

	b := &storage.Blob{
		Name:         blobName,
		LastUpdated:  protocompat.TimestampNow(),
		ModifiedTime: protocompat.TimestampNow(),
		Length:       zipF.FileInfo().Size(),
	}

	return s.datastore.Upsert(s.ctx, b, r)
}

func (s *handlerTestSuite) upsertV4ZipFile(zipPath string) error {
	zipR, err := zip.OpenReader(zipPath)
	s.Require().NoError(err)
	defer utils.IgnoreError(zipR.Close)
	for _, zipF := range zipR.File {
		if strings.HasPrefix(zipF.Name, scannerV4DefsPrefix) {
			err = s.upsertBlob(zipF, offlineScannerV4DefsBlobName)
			s.Require().NoError(err)
			return nil
		}
	}
	return errors.New("test failed")
}

func (s *handlerTestSuite) TestServeHTTP_Invalid() {
	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// PUT is not allowed.
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPut, "https://central.stackrox.svc/scannerdefinitions", nil)
	s.Require().NoError(err)
	h.ServeHTTP(w, req)
	s.Equal(http.StatusMethodNotAllowed, w.Result().StatusCode)

	// There are no query params to identify the file to GET.
	req, err = http.NewRequestWithContext(s.ctx, http.MethodGet, "https://central.stackrox.svc/scannerdefinitions", nil)
	s.Require().NoError(err)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusBadRequest, w.Result().StatusCode)

	// There is no request body to POST.
	req, err = http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", nil)
	s.Require().NoError(err)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusBadRequest, w.Result().StatusCode)
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Post_V2() {
	s.T().Setenv(env.OfflineModeEnv.EnvVar(), "true")
	s.T().Setenv(features.ScannerV4.EnvVar(), "false")

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	req := s.postRequestV2()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode)
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Get_V2() {
	s.T().Setenv(env.OfflineModeEnv.EnvVar(), "true")
	s.T().Setenv(features.ScannerV4.EnvVar(), "false")

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// No scanner-defs found.
	getReq := s.getRequestUUID()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusNotFound, w.Result().StatusCode)

	// Post scanner-defs.
	postReq := s.postRequestV2()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, postReq)
	s.Require().Equal(http.StatusOK, w.Result().StatusCode)

	// Bad request after data is uploaded should give offline data.
	getReq = s.getRequestBadUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal("application/zip", w.Result().Header.Get("Content-Type"))

	// Get offline data again with good UUID.
	getReq = s.getRequestUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal("application/zip", w.Result().Header.Get("Content-Type"))
	s.Greater(w.Body.Len(), 0)

	// Should get file from offline data.
	getReq = s.getRequestUUIDAndFile("manifest.json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal("application/json", w.Result().Header.Get("Content-Type"))
	s.Equal(v2ManifestContent, w.Body.String())
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V2() {
	s.T().Setenv(features.ScannerV4.EnvVar(), "false")

	// As great as it would be to test with real data,
	// it's more reliable to keep everything local,
	// so start a local server to mimic definitions.stackrox.io.
	s.startMockDefinitionsStackRoxIO()

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// Should not get anything with bad UUID.
	req := s.getRequestBadUUID()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Result().StatusCode)

	// Should get online vulns.
	req = s.getRequestUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode, w.Body.String())
	s.Equal("application/zip", w.Result().Header.Get("Content-Type"))
	s.Greater(w.Body.Len(), 0)

	// Should get the specified file from online update.
	req = s.getRequestUUIDAndFile("manifest.json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode, w.Body.String())
	s.Equal("application/json", w.Result().Header.Get("Content-Type"))
	s.Equal(v2ManifestContent, w.Body.String())

	// Write offline definitions, directly.
	// Set the offline dump's modified time to later than the online update's.
	s.mustWriteV2Blob(content1, time.Now().Add(time.Hour))

	// Serve the offline dump, as it is more recent.
	req = s.getRequestUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode, w.Body.String())
	s.Equal(content1, w.Body.String())

	// Set the offline dump's modified time to earlier than the online update's.
	s.mustWriteV2Blob(content2, nov23)

	// Serve the online dump, as it is now more recent.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode, w.Body.String())
	s.NotEqual(content1, w.Body.String())
	s.NotEqual(content2, w.Body.String())

	// File is unmodified.
	req.Header.Set(ifModifiedSinceHeader, time.Now().UTC().Format(http.TimeFormat))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotModified, w.Result().StatusCode, w.Body.String())
	s.Empty(w.Body.String())
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Post_V4() {
	s.T().Setenv(env.OfflineModeEnv.EnvVar(), "true")
	s.T().Setenv(features.ScannerV4.EnvVar(), "true")

	s.T().Run("single v4 definition", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()
		prev := mainVersionVariants
		mainVersionVariants = map[string]bool{"development": true}
		t.Cleanup(func() {
			mainVersionVariants = prev
		})
		req := s.postRequestV4(newZipBuilder().
			addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
			addFile(s, "v4-definitions-dev.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
  "version": "dev",
  "release_versions": "development"
}`)).
				buildBuffer(s).Bytes()).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusOK, w.Result().StatusCode, "body: %s", w.Body.String())
	})
	s.T().Run("missing v4 definition", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()

		req := s.postRequestV4(newZipBuilder().
			addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusBadRequest, w.Result().StatusCode, "body: %s", w.Body.String())
		s.Contains(w.Body.String(), "the uploaded bundle is incompatible with release version number")
	})
	s.T().Run("missing v2 definition", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()

		req := s.postRequestV4(newZipBuilder().
			addFile(s, "v4-definitions-dev.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
  "version": "dev",
  "release_versions": "development"
}`)).
				buildBuffer(s).Bytes()).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusBadRequest, w.Result().StatusCode, "body: %s", w.Body.String())
		s.Contains(w.Body.String(), "the uploaded bundle is incompatible with release version number")
	})
	s.T().Run("v4 definition with unsupported release", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()
		prev := mainVersionVariants
		mainVersionVariants = map[string]bool{"development": true}
		t.Cleanup(func() {
			mainVersionVariants = prev
		})
		req := s.postRequestV4(newZipBuilder().
			addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
			addFile(s, "v4-definitions-dev.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
  "version": "dev",
  "release_versions": "unsupported release"
}`)).
				buildBuffer(s).Bytes()).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusBadRequest, w.Result().StatusCode, "body: %s", w.Body.String())
		s.Contains(w.Body.String(), "the uploaded bundle is incompatible with release version number")
	})
	s.T().Run("latest bundle with multiple v4 definitions", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()
		prev := mainVersionVariants
		mainVersionVariants = map[string]bool{"development": true}
		t.Cleanup(func() {
			mainVersionVariants = prev
		})

		req := s.postRequestV4(newZipBuilder().
			addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
			addFile(s, "v4-definitions-v1.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
					  "version": "v1",
					  "release_versions": "unsupported"
					}`)).
				buildBuffer(s).Bytes()).
			addFile(s, "v4-definitions-v2.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
					  "version": "v2",
					  "release_versions": "development"
					}`)).
				addFile(s, "vulnerabilities.zip", "Scanner V4 vulnerabilities", []byte(content2)).
				buildBuffer(s).Bytes()).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusOK, w.Result().StatusCode, "body: %s", w.Body.String())

		req = s.getRequestVersion("v2")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Result().StatusCode, w.Body.String())

	})
	s.T().Run("latest bundle with multiple v4 definitions without supported", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()
		prev := mainVersionVariants
		mainVersionVariants = map[string]bool{"development": true}
		t.Cleanup(func() {
			mainVersionVariants = prev
		})

		req := s.postRequestV4(newZipBuilder().
			addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
			addFile(s, "v4-definitions-v1.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
					  "version": "v1",
					  "release_versions": "unsupported"
					}`)).buildBuffer(s).Bytes()).
			addFile(s, "v4-definitions-v2.zip", "Scanner V4 content", newZipBuilder().
				addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
					  "version": "v2",
					  "release_versions": "another unsupported"
					}`)).buildBuffer(s).Bytes()).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusBadRequest, w.Result().StatusCode, "body: %s", w.Body.String())
		s.Contains(w.Body.String(), "the uploaded bundle is incompatible with release version number")
	})
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Get_V4() {
	s.T().Setenv(env.OfflineModeEnv.EnvVar(), "true")
	s.T().Setenv(features.ScannerV4.EnvVar(), "true")

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// No scanner defs found.
	req := s.getRequestVersion("4.5.0")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Result().StatusCode)

	// No mapping json file
	req = s.getRequestFile("name2repos")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Result().StatusCode)

	// No mapping json file
	req = s.getRequestFile("repo2cpe")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Result().StatusCode)

	filePath := filepath.Join(s.T().TempDir(), "test.zip")
	outFile, err := os.Create(filePath)
	s.Require().NoError(err)
	_, err = io.Copy(outFile, newZipBuilder().
		addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
		addFile(s, "scanner-v4-defs-4.5.zip", "Scanner V4 4.5", newZipBuilder().
			addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
					  "version": "4.5",
					  "release_versions": "some-release"
					}`)).buildBuffer(s).Bytes()).
		addFile(s, "v4-definitions-v2.zip", "Scanner V4 v2", newZipBuilder().
			addFile(s, "manifest.json", "Scanner V4 manifest", []byte(`{
					  "version": "v2",
					  "release_versions": "some-release"
					}`)).
			addFile(s, "vulnerabilities.zip", "Scanner V4 vulnerabilities", []byte(content2)).
			addFile(s, "container-name-repos-map.json", "Scanner V4 repo-to-name map", []byte(content1)).
			addFile(s, "repository-to-cpe.json", "Scanner V4 repo-to-cpe map", []byte(`{}`)).
			buildBuffer(s).Bytes()).
		buildBuffer(s))
	s.Require().NoError(err)
	utils.IgnoreError(outFile.Close)

	// Upload offline vulns, directly.
	err = s.upsertV4ZipFile(filePath)
	s.Require().NoError(err)

	s.T().Run("get 4.5", func(t *testing.T) {
		req = s.getRequestVersion("4.5.0")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		// This fails on release builds because checks don't happen on dev builds.
		if buildinfo.ReleaseBuild {
			s.Equalf(http.StatusNotFound, w.Result().StatusCode, "body: %s", w.Body.String())
		} else {
			s.Equal(http.StatusOK, w.Result().StatusCode, "body: %s", w.Body.String())
			s.Equal(content2, w.Body.String())
		}
	})

	s.T().Run("get v2", func(t *testing.T) {
		req = s.getRequestVersion("v2")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		// This fails on release builds because checks don't happen on dev builds.
		s.Equal(http.StatusOK, w.Result().StatusCode)
	})

	s.T().Run("get repo2cpe", func(t *testing.T) {
		req = s.getRequestFile("repo2cpe")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Result().StatusCode)
		s.Equal("application/json", w.Result().Header.Get("Content-Type"))
		s.Greater(w.Body.Len(), 0)
		s.Equal(`{}`, w.Body.String())
	})

	s.T().Run("get name2repos", func(t *testing.T) {
		req = s.getRequestFile("name2repos")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Result().StatusCode)
		s.Greater(w.Body.Len(), 0)
		s.Equal(content1, w.Body.String())
	})

	s.T().Run("get invalid", func(t *testing.T) {
		req = s.getRequestFile("invalid")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusNotFound, w.Result().StatusCode)
	})
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V4() {
	// As great as it would be to test with real data,
	// it's more reliable to keep everything local,
	// so start a local server to mimic definitions.stackrox.io.
	s.startMockDefinitionsStackRoxIO()

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	s.T().Run("not found", func(t *testing.T) {
		req := s.getRequestVersion("randomName")
		h.ServeHTTP(w, req)
		s.Equal(http.StatusNotFound, w.Result().StatusCode)
	})
	s.T().Run("should get dev zip file from online update", func(t *testing.T) {
		req := s.getRequestVersion(v4Dev)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Result().StatusCode)
		s.Equal("application/zip", w.Result().Header.Get("Content-Type"))
		s.Greater(w.Body.Len(), 0)
	})
	s.T().Run("release version", func(t *testing.T) {
		req := s.getRequestVersion(v4V1)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Result().StatusCode)
		s.Equal("application/zip", w.Result().Header.Get("Content-Type"))
		s.Greater(w.Body.Len(), 0)
	})
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V4_Mappings() {
	// As great as it would be to test with real data,
	// it's more reliable to keep everything local,
	// so start a local server to mimic definitions.stackrox.io.
	s.startMockDefinitionsStackRoxIO()

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// Nothing should be found
	req := s.getRequestFile("randomName")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Result().StatusCode)

	// Should get mapping json file from online update.
	req = s.getRequestFile("name2repos")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal("application/json", w.Result().Header.Get("Content-Type"))
	s.Equal(name2repos, w.Body.String())

	// Should get mapping json file from online update.
	req = s.getRequestFile("repo2cpe")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal("application/json", w.Result().Header.Get("Content-Type"))
	s.Equal(repo2cpe, w.Body.String())
}
