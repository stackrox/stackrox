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

	v2ManifestContent = `{
  "since": "yesterday",
  "until": "today"
}`
)

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

func (s *handlerTestSuite) postRequestV2() *http.Request {
	var manifestBuf bytes.Buffer
	zw := zip.NewWriter(&manifestBuf)
	file, err := zw.CreateHeader(&zip.FileHeader{
		Name:               "manifest.json",
		Comment:            "Scanner V2 manifest",
		UncompressedSize64: uint64(len(v2ManifestContent)),
	})
	s.Require().NoError(err)
	_, err = file.Write([]byte(v2ManifestContent))
	s.Require().NoError(err)
	s.Require().NoError(zw.Close())

	var buf bytes.Buffer
	zw = zip.NewWriter(&buf)
	file, err = zw.CreateHeader(&zip.FileHeader{
		Name:               "scanner-defs.zip",
		Comment:            "Scanner V2 content",
		UncompressedSize64: uint64(manifestBuf.Len()),
	})
	s.Require().NoError(err)
	_, err = file.Write(manifestBuf.Bytes())
	s.Require().NoError(err)
	s.Require().NoError(zw.Close())

	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", &buf)
	s.Require().NoError(err)

	return req
}

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

func (s *handlerTestSuite) postRequestV4(body io.Reader) *http.Request {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", body)
	s.Require().NoError(err)
	return req
}

func (s *handlerTestSuite) mustWriteBlob(content string, modTime time.Time) {
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
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=e799c68a-671f-44db-9682-f24248cd0ffe"
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
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?uuid=e799c68a-671f-44db-9682-f24248cd0ffe&file=%s", file)
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

func (s *handlerTestSuite) mockHandleDefsFile(zipF *zip.File, blobName string) error {
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

func (s *handlerTestSuite) mockHandleZipContents(zipPath string) error {
	zipR, err := zip.OpenReader(zipPath)
	s.Require().NoError(err)
	defer utils.IgnoreError(zipR.Close)
	for _, zipF := range zipR.File {
		if strings.HasPrefix(zipF.Name, scannerV4DefsPrefix) {
			err = s.mockHandleDefsFile(zipF, offlineScannerV4DefsBlobName)
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
	s.Equal(http.StatusMethodNotAllowed, w.Code)

	// There are no query params to identify the file to GET.
	req, err = http.NewRequestWithContext(s.ctx, http.MethodGet, "https://central.stackrox.svc/scannerdefinitions", nil)
	s.Require().NoError(err)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusBadRequest, w.Code)

	// There is no request body to POST.
	req, err = http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", nil)
	s.Require().NoError(err)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Post_V2() {
	s.T().Setenv(env.OfflineModeEnv.EnvVar(), "true")
	s.T().Setenv(features.ScannerV4.EnvVar(), "false")

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	req := s.postRequestV2()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Get_V2() {
	s.T().Setenv(env.OfflineModeEnv.EnvVar(), "true")
	s.T().Setenv(features.ScannerV4.EnvVar(), "false")

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// No scanner-defs found.
	getReq := s.getRequestUUID()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusNotFound, w.Code)

	// Post scanner-defs.
	postReq := s.postRequestV2()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, postReq)
	s.Require().Equal(http.StatusOK, w.Code)

	// Bad request after data is uploaded should give offline data.
	getReq = s.getRequestBadUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zip", w.Header().Get("Content-Type"))

	// Get offline data again with good UUID.
	getReq = s.getRequestUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zip", w.Header().Get("Content-Type"))
	s.Greater(w.Body.Len(), 0)

	// Should get file from offline data.
	getReq = s.getRequestUUIDAndFile("manifest.json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
	s.Equal(v2ManifestContent, w.Body.String())
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V2() {
	s.T().Setenv(features.ScannerV4.EnvVar(), "false")

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// Should not get anything with bad UUID.
	req := s.getRequestBadUUID()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

	// Should get online vulns.
	req = s.getRequestUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zip", w.Header().Get("Content-Type"))
	s.Greater(w.Body.Len(), 0)

	// Should get file from online update.
	req = s.getRequestUUIDAndFile("manifest.json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
	s.Regexp(`{"since":".*","until":".*"}`, w.Body.String())

	// Write offline definitions, directly.
	// Set the offline dump's modified time to later than the online update's.
	s.mustWriteBlob(content1, time.Now().Add(time.Hour))

	// Serve the offline dump, as it is more recent.
	req = s.getRequestUUID()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal(content1, w.Body.String())

	// Set the offline dump's modified time to earlier than the online update's.
	s.mustWriteBlob(content2, nov23)

	// Serve the online dump, as it is now more recent.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.NotEqual(content2, w.Body.String())

	// File is unmodified.
	req.Header.Set(ifModifiedSinceHeader, time.Now().UTC().Format(http.TimeFormat))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotModified, w.Code)
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
		s.Equalf(http.StatusOK, w.Code, "body: %s", w.Body.String())
	})
	s.T().Run("missing v4 definition", func(t *testing.T) {
		h := New(s.datastore, handlerOpts{})
		w := httptest.NewRecorder()

		req := s.postRequestV4(newZipBuilder().
			addFile(s, "scanner-defs.zip", "Scanner V2 content", []byte(content1)).
			buildBuffer(s))
		h.ServeHTTP(w, req)
		s.Equalf(http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
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
		s.Equalf(http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
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
		s.Equalf(http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
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
		s.Equalf(http.StatusOK, w.Code, "body: %s", w.Body.String())

		req = s.getRequestVersion("v2")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)

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
		s.Equalf(http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
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
	s.Equal(http.StatusNotFound, w.Code)

	// No mapping json file
	req = s.getRequestFile("name2repos")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

	// No mapping json file
	req = s.getRequestFile("repo2cpe")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

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
	err = s.mockHandleZipContents(filePath)
	s.Require().NoError(err)

	s.T().Run("get 4.5", func(t *testing.T) {
		req = s.getRequestVersion("4.5.0")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		// This fails on release builds because checks don't happen on dev builds.
		if buildinfo.ReleaseBuild {
			s.Equalf(http.StatusNotFound, w.Code, "body: %s", w.Body.String())
		} else {
			s.Equal(http.StatusOK, w.Code, "body: %s", w.Body.String())
			s.Equal(content2, w.Body.String())
		}
	})

	s.T().Run("get v2", func(t *testing.T) {
		req = s.getRequestVersion("v2")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		// This fails on release builds because checks don't happen on dev builds.
		s.Equal(http.StatusOK, w.Code)
	})

	s.T().Run("get repo2cpe", func(t *testing.T) {
		req = s.getRequestFile("repo2cpe")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
		s.Equal("application/json", w.Header().Get("Content-Type"))
		s.Greater(w.Body.Len(), 0)
		s.Equal(`{}`, w.Body.String())
	})

	s.T().Run("get name2repos", func(t *testing.T) {
		req = s.getRequestFile("name2repos")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
		s.Greater(w.Body.Len(), 0)
		s.Equal(content1, w.Body.String())
	})

	s.T().Run("get invalid", func(t *testing.T) {
		req = s.getRequestFile("invalid")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusNotFound, w.Code)
	})
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V4() {
	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	s.T().Run("not found", func(t *testing.T) {
		req := s.getRequestVersion("randomName")
		h.ServeHTTP(w, req)
		s.Equal(http.StatusNotFound, w.Code)
	})
	s.T().Run("should get dev zstd file from online update", func(t *testing.T) {
		req := s.getRequestVersion("dev")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
		s.Equal("application/zip", w.Header().Get("Content-Type"))
		s.Greater(w.Body.Len(), 0)
	})
	s.T().Run("release version", func(t *testing.T) {
		req := s.getRequestVersion("v1")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
		s.Equal("application/zip", w.Header().Get("Content-Type"))
		s.Greater(w.Body.Len(), 0)
	})
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V4_Mappings() {
	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// Nothing should be found
	req := s.getRequestFile("randomName")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

	// Should get mapping json file from online update.
	req = s.getRequestFile("name2repos")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))

	// Should get mapping json file from online update.
	req = s.getRequestFile("repo2cpe")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
}
