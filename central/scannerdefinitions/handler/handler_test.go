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
	v4ManifestContent = `{
  "version": "dev"
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

func (s *handlerTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
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

func (s *handlerTestSuite) postRequestV4() *http.Request {
	// V4 ZIP file contents.
	var v4Buf bytes.Buffer
	zw := zip.NewWriter(&v4Buf)
	file, err := zw.CreateHeader(&zip.FileHeader{
		Name:               "manifest.json",
		Comment:            "Scanner V4 manifest",
		UncompressedSize64: uint64(len(v4ManifestContent)),
	})
	s.Require().NoError(err)
	_, err = file.Write([]byte(v4ManifestContent))
	s.Require().NoError(err)
	s.Require().NoError(zw.Close())

	var buf bytes.Buffer
	zw = zip.NewWriter(&buf)

	// Currently, we need both V2 and V4 files when Scanner V4 is enabled.
	file, err = zw.CreateHeader(&zip.FileHeader{
		Name:               "scanner-defs.zip",
		Comment:            "Scanner V2 content",
		UncompressedSize64: uint64(len(content1)),
	})
	s.Require().NoError(err)
	_, err = file.Write([]byte(content1))
	s.Require().NoError(err)

	file, err = zw.CreateHeader(&zip.FileHeader{
		Name:               "scanner-v4-defs.zip",
		Comment:            "Scanner V4 content",
		UncompressedSize64: uint64(v4Buf.Len()),
	})
	s.Require().NoError(err)
	_, err = file.Write(v4Buf.Bytes())
	s.Require().NoError(err)

	s.Require().NoError(zw.Close())

	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, "https://central.stackrox.svc/scannerdefinitions", &buf)
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
	// TODO: This should be a 404. Update in a followup.
	s.Equal(http.StatusInternalServerError, w.Code)

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
	s.Regexpf(`{"since":".*","until":".*"}`, w.Body.String(), "content1 did not match")

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

	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	req := s.postRequestV4()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
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
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

	// No mapping json file
	req = s.getRequestFile("repo2cpe")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

	url := "https://storage.googleapis.com/scanner-support-public/offline/v1/4.5/scanner-vulns-4.5.zip"
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, url, nil)
	s.Require().NoError(err)
	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer utils.IgnoreError(resp.Body.Close)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	filePath := filepath.Join(s.T().TempDir(), "test.zip")
	outFile, err := os.Create(filePath)
	s.Require().NoError(err)

	_, err = io.Copy(outFile, resp.Body)
	s.Require().NoError(err)
	utils.IgnoreError(outFile.Close)

	err = s.mockHandleZipContents(filePath)
	s.Require().NoError(err)

	// This will fail because 4.5.0 uses the multi-bundle ZIP format.
	req = s.getRequestVersion("4.5.0")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	// Set the header properly.
	req.Header.Set("X-Scanner-V4-Accept", "application/vnd.stackrox.scanner-v4.multi-bundle+zip")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zip", w.Header().Get("Content-Type"))

	w = httptest.NewRecorder()
	req = s.getRequestFile("repo2cpe")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))

	w = httptest.NewRecorder()
	req = s.getRequestFile("name2repos")
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get_V4() {
	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	req := s.getRequestVersion("randomName")
	h.ServeHTTP(w, req)
	// If the version is invalid or versioned bundle cannot be found, it's a 500
	s.Equal(http.StatusInternalServerError, w.Code)

	// Should get dev zstd file from online update.
	req = s.getRequestVersion("dev")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zstd", w.Header().Get("Content-Type"))

	// Release version.
	req = s.getRequestVersion("4.4.0")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zstd", w.Header().Get("Content-Type"))

	// Should get dev zstd file from online update.
	req = s.getRequestVersion("4.3.x-nightly-20240106")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zstd", w.Header().Get("Content-Type"))

	// Multi-bundle ZIP.
	req = s.getRequestVersion("dev")
	req.Header.Set("X-Scanner-V4-Accept", "application/vnd.stackrox.scanner-v4.multi-bundle+zip")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/zip", w.Header().Get("Content-Type"))
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
