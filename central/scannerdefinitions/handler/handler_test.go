//go:build sql_integration

package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/mock"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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
	s.LessOrEqual(len(entries), 2)
	if len(entries) == 2 {
		s.True(strings.HasPrefix(entries[0].Name(), definitionsBaseDir))
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
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?type=%s", file)
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
	w := mock.NewResponseWriter()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Add scanner defs.
	s.mustWriteOffline(content1, time.Now())

	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, content1, w.Data.String())
}

func (s *handlerTestSuite) TestServeHTTP_Online_Get() {
	t := s.T()
	h := New(s.datastore, handlerOpts{})

	w := mock.NewResponseWriter()

	// Should not get anything.
	req := s.mustGetBadRequest(t)
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Should get file from online update.
	req = s.mustGetRequestWithFile(t, "manifest.json")
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Regexpf(t, `{"since":".*","until":".*"}`, w.Data.String(), "content1 did not match")
	// Should get online update.
	req = s.mustGetRequest(t)
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Write offline definitions.
	s.mustWriteOffline(content1, time.Now())

	// Set the offline dump's modified time to later than the online update's.
	s.mustWriteOffline(content1, time.Now().Add(time.Hour))

	// Served the offline dump, as it is more recent.
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, content1, w.Data.String())

	// Set the offline dump's modified time to earlier than the online update's.
	s.mustWriteOffline(content2, nov23)

	// Serve the online dump, as it is now more recent.
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, content2, w.Data.String())

	// File is unmodified.
	req.Header.Set(ifModifiedSinceHeader, time.Now().UTC().Format(http.TimeFormat))
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotModified, w.Code)
	assert.Empty(t, w.Data.String())
}

func (s *handlerTestSuite) TestServeHTTP_Online_Mappings_Get() {
	t := s.T()
	h := New(s.datastore, handlerOpts{})

	w := mock.NewResponseWriter()

	// Nothing should be found
	req := s.getRequestWithJSONFile(t, "randomName")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Should get mapping json file from online update.
	req = s.mustGetRequestWithFile(t, "name2cpe")
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Should get mapping json file from online update.
	req = s.mustGetRequestWithFile(t, "repo2cpe")
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func (s *handlerTestSuite) mustWriteOffline(content string, modTime time.Time) {
	modifiedTime, err := types.TimestampProto(modTime)
	s.Require().NoError(err)
	blob := &storage.Blob{
		Name:         offlineScannerDefinitionBlobName,
		Length:       int64(len(content)),
		ModifiedTime: modifiedTime,
		LastUpdated:  types.TimestampNow(),
	}
	s.Require().NoError(s.datastore.Upsert(s.ctx, blob, bytes.NewBuffer([]byte(content))))
}
