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
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	content1 = "Hello, world!"
	content2 = "Papaya"
)

type handlerTestSuite struct {
	suite.Suite
	ctx       context.Context
	datastore datastore.Datastore
	testDB    *pgtest.TestPostgres
	tmpDir    string
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

func (s *handlerTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	blobStore := store.New(s.testDB.DB)
	s.datastore = datastore.NewDatastore(blobStore, nil)
	var err error
	s.tmpDir, err = os.MkdirTemp("", "handler-test")
	s.Require().NoError(err)
	s.T().Setenv("TMPDIR", s.tmpDir)
}

func (s *handlerTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *handlerTestSuite) TearDownSuite() {
	entries, err := os.ReadDir(s.tmpDir)
	s.NoError(err)
	s.LessOrEqual(len(entries), 3)
	if len(entries) == 3 {
		s.True(strings.HasPrefix(entries[0].Name(), definitionsBaseDir))
		s.True(strings.HasPrefix(entries[1].Name(), definitionsBaseDir))
		s.True(strings.HasPrefix(entries[2].Name(), definitionsBaseDir))
	}

	s.testDB.Teardown(s.T())
	utils.IgnoreError(func() error { return os.RemoveAll(s.tmpDir) })
}

func (s *handlerTestSuite) mustGetRequest(t *testing.T) *http.Request {
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=e799c68a-671f-44db-9682-f24248cd0ffe"
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)

	require.NoError(t, err)

	return req
}

func (s *handlerTestSuite) getRequestWithJSONFile(t *testing.T, file string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?file=%s", file)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func (s *handlerTestSuite) getRequestWithVersionedFile(t *testing.T, v string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?version=%s", v)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func (s *handlerTestSuite) mustGetRequestWithFile(t *testing.T, file string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?uuid=e799c68a-671f-44db-9682-f24248cd0ffe&file=%s", file)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func (s *handlerTestSuite) mustGetBadRequest(t *testing.T) *http.Request {
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=fail"
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func (s *handlerTestSuite) TestServeHTTP_Offline_Get() {
	t := s.T()
	t.Setenv(env.OfflineModeEnv.EnvVar(), "true")

	h := New(s.datastore, handlerOpts{})

	// No scanner defs found.
	req := s.mustGetRequest(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Add scanner defs.
	s.mustWriteOffline(content1, time.Now())

	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, content1, w.Body.String())
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get() {
	t := s.T()
	h := New(s.datastore, handlerOpts{})

	// Should not get anything.
	req := s.mustGetBadRequest(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Should get file from online update.
	req = s.mustGetRequestWithFile(t, "manifest.json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Regexpf(t, `{"since":".*","until":".*"}`, w.Body.String(), "content1 did not match")

	// Should get online update.
	req = s.mustGetRequest(t)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Write offline definitions.
	s.mustWriteOffline(content1, time.Now())

	// Set the offline dump's modified time to later than the online update's.
	s.mustWriteOffline(content1, time.Now().Add(time.Hour))

	// Served the offline dump, as it is more recent.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, content1, w.Body.String())

	// Set the offline dump's modified time to earlier than the online update's.
	s.mustWriteOffline(content2, nov23)

	// Serve the online dump, as it is now more recent.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, content2, w.Body.String())

	// File is unmodified.
	req.Header.Set(ifModifiedSinceHeader, time.Now().UTC().Format(http.TimeFormat))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotModified, w.Code)
	assert.Empty(t, w.Body.String())
}

func (s *handlerTestSuite) TestServeHTTP_Online_ZSTD_Bundle_Get() {
	t := s.T()
	h := New(s.datastore, handlerOpts{})

	w := httptest.NewRecorder()
	req := s.getRequestWithVersionedFile(t, "randomName")
	h.ServeHTTP(w, req)
	// If the version is invalid or versioned bundle cannot be found, it's a 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Should get dev zstd file from online update.
	req = s.getRequestWithVersionedFile(t, "dev")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zstd", w.Header().Get("Content-Type"))

	req = s.getRequestWithVersionedFile(t, "4.3.x-173-g6bbb2e07dc")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zstd", w.Header().Get("Content-Type"))

	// Should get dev zstd file from online update.
	req = s.getRequestWithVersionedFile(t, "4.3.x-nightly-20240106")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zstd", w.Header().Get("Content-Type"))
}

func (s *handlerTestSuite) TestServeHTTP_Online_Mappings_Get() {
	t := s.T()
	h := New(s.datastore, handlerOpts{})

	w := httptest.NewRecorder()

	// Nothing should be found
	req := s.getRequestWithJSONFile(t, "randomName")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Should get mapping json file from online update.
	req = s.getRequestWithJSONFile(t, "name2repos")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Should get mapping json file from online update.
	req = s.getRequestWithJSONFile(t, "repo2cpe")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func (s *handlerTestSuite) mustWriteOffline(content string, modTime time.Time) {
	modifiedTime, err := protocompat.ConvertTimeToTimestampOrError(modTime)
	s.Require().NoError(err)
	blob := &storage.Blob{
		Name:         offlineScannerDefinitionBlobName,
		Length:       int64(len(content)),
		ModifiedTime: modifiedTime,
		LastUpdated:  protocompat.TimestampNow(),
	}
	s.Require().NoError(s.datastore.Upsert(s.ctx, blob, bytes.NewBuffer([]byte(content))))
}

func (s *handlerTestSuite) TestServeHTTP_v4_Offline_Get() {
	t := s.T()
	t.Setenv(env.OfflineModeEnv.EnvVar(), "true")
	h := New(s.datastore, handlerOpts{})
	w := httptest.NewRecorder()

	// No scanner defs found.
	req := s.getRequestWithVersionedFile(t, "4.3.0")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// No mapping json file
	req = s.getRequestWithJSONFile(t, "name2repos")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// No mapping json file
	req = s.getRequestWithJSONFile(t, "repo2cpe")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	tempDir := t.TempDir()
	filePath := tempDir + "/test.zip"

	url := "https://storage.googleapis.com/scanner-support-public/offline/v1/4.3/scanner-vulns-4.3.zip"
	resp, err := http.Get(url)
	// Skip the test if file cannot be downloaded or status code is not OK.
	if err != nil {
		return
	}
	defer utils.IgnoreError(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		return
	}

	outFile, err := os.Create(filePath)
	s.Require().NoError(err)

	_, err = io.Copy(outFile, resp.Body)
	s.Require().NoError(err)
	utils.IgnoreError(outFile.Close)

	err = s.mockHandleZipContents(filePath)
	s.Require().NoError(err)

	req = s.getRequestWithVersionedFile(t, "4.3.0")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zstd", w.Header().Get("Content-Type"))

	w = httptest.NewRecorder()
	req = s.getRequestWithJSONFile(t, "repo2cpe")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	w = httptest.NewRecorder()
	req = s.getRequestWithJSONFile(t, "name2repos")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
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
			err = s.mockHandleDefsFile(zipF, offlineScannerV4DefinitionBlobName)
			s.Require().NoError(err)
			return nil
		}
	}
	return errors.New("test failed")
}
