package handler

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/pkg/httputil/mock"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

func (s *handlerTestSuite) mustGetRequestWithParams(t *testing.T, k, v string) *http.Request {
	centralURL := "https://central.stackrox.svc/scanner-v4/testrepomapping"
	u, err := url.Parse(centralURL)
	require.NoError(t, err)

	// Create and set query parameters
	params := url.Values{}
	params.Add(k, v) // Replace 'key' and 'value' with your actual key-value pair
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, u.String(), nil)
	require.NoError(t, err)

	return req
}

func (s *handlerTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	blobStore := store.New(s.testDB.DB)
	s.datastore = datastore.NewDatastore(blobStore, nil)
	var err error
	s.tmpDir, err = os.MkdirTemp("", "handler-test")
	s.Require().NoError(err)
}

func (s *handlerTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE blobs CASCADE")
	s.T().Log("blobs", tag)
	s.NoError(err)
}

func (s *handlerTestSuite) TearDownSuite() {
	entries, err := os.ReadDir(s.tmpDir)
	s.NoError(err)

	s.LessOrEqual(len(entries), 1)

	if len(entries) == 1 {
		s.True(strings.HasPrefix(entries[0].Name(), baseDir))
	}

	s.testDB.Teardown(s.T())
	utils.IgnoreError(func() error { return os.RemoveAll(s.tmpDir) })
}

func (s *handlerTestSuite) TestServeHTTP_400() {
	t := s.T()
	// Set the environment variable
	err := os.Setenv("ROX_SCANNER_V4_CVSS_MAX_INITIAL_WAIT", "1ms")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	h := New(s.datastore)
	time.Sleep(3 * time.Second)
	w := mock.NewResponseWriter()
	req1 := s.mustGetRequestWithParams(t, "file", "random")
	h.ServeHTTP(w, req1)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	req2 := s.mustGetRequestWithParams(t, "", "")
	h.ServeHTTP(w, req2)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func (s *handlerTestSuite) TestServeHTTP_Get() {
	t := s.T()
	t.Setenv("ROX_MAPPING_UPDATE_MAX_INITIAL_WAIT", "1ms")

	h := New(s.datastore)
	time.Sleep(3 * time.Second)
	w := mock.NewResponseWriter()
	req := s.mustGetRequestWithParams(t, "file", "name2cpe")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
