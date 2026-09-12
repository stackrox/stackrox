package cli

import (
	"archive/tar"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func writeTarball(t *testing.T, dir, tarName, entryName string, content []byte) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, tarName+".tar.gz"))
	if err != nil {
		t.Fatal(err)
	}

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: entryName,
		Size: int64(len(content)),
		Mode: 0755,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

type HandlerSuite struct {
	suite.Suite
	dir     string
	handler http.HandlerFunc
	payload []byte
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

func (s *HandlerSuite) SetupTest() {
	s.dir = s.T().TempDir()
	s.handler = handlerWithDir(s.dir)
	s.payload = []byte("fake roxctl binary content")
	writeTarball(s.T(), s.dir, "roxctl-linux-amd64", "roxctl-linux-amd64", s.payload)
	// Windows: tarball strips .exe from the name (required by release pipeline),
	// but the entry inside the tarball preserves .exe.
	writeTarball(s.T(), s.dir, "roxctl-windows-amd64", "roxctl-windows-amd64.exe", s.payload)
}

func (s *HandlerSuite) TestServesLinuxBinaryViaGET() {
	req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-linux-amd64", nil)
	rr := httptest.NewRecorder()
	s.handler(rr, req)
	s.Equal(http.StatusOK, rr.Code)
	s.Equal(`attachment; filename="roxctl-linux-amd64"`, rr.Header().Get("Content-Disposition"))
	s.Equal("application/octet-stream", rr.Header().Get("Content-Type"))
	s.Equal("26", rr.Header().Get("Content-Length"))
	s.Equal("none", rr.Header().Get("Accept-Ranges"))
	s.Equal("no-cache", rr.Header().Get("Cache-Control"))
	s.Equal(s.payload, rr.Body.Bytes())
}

func (s *HandlerSuite) TestServesWindowsBinaryViaGET() {
	req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-windows-amd64.exe", nil)
	rr := httptest.NewRecorder()
	s.handler(rr, req)
	s.Equal(http.StatusOK, rr.Code)
	s.Equal(`attachment; filename="roxctl-windows-amd64.exe"`, rr.Header().Get("Content-Disposition"))
	s.Equal(s.payload, rr.Body.Bytes())
}

func (s *HandlerSuite) TestReturnsHeadersButNoBodyForHEAD() {
	req := httptest.NewRequest(http.MethodHead, "/api/cli/download/roxctl-linux-amd64", nil)
	rr := httptest.NewRecorder()
	s.handler(rr, req)
	s.Equal(http.StatusOK, rr.Code)
	s.Equal("26", rr.Header().Get("Content-Length"))
	s.Empty(rr.Body.Bytes())
}

func (s *HandlerSuite) TestReturns404ForMissingTarball() {
	req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-darwin-amd64", nil)
	rr := httptest.NewRecorder()
	s.handler(rr, req)
	s.Equal(http.StatusNotFound, rr.Code)
}

func (s *HandlerSuite) TestReturns404WhenBinaryNotInTarball() {
	dir := s.T().TempDir()
	// Tarball exists but entry name does not match the requested filename.
	writeTarball(s.T(), dir, "roxctl-linux-amd64", "wrong-entry-name", s.payload)
	handler := handlerWithDir(dir)
	req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-linux-amd64", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	s.Equal(http.StatusNotFound, rr.Code)
}

func (s *HandlerSuite) TestReturns405ForUnsupportedMethod() {
	req := httptest.NewRequest(http.MethodPost, "/api/cli/download/roxctl-linux-amd64", nil)
	rr := httptest.NewRecorder()
	s.handler(rr, req)
	s.Equal(http.StatusMethodNotAllowed, rr.Code)
	s.Equal("GET, HEAD", rr.Header().Get("Allow"))
}

func (s *HandlerSuite) TestReturns500ForCorruptedGzip() {
	dir := s.T().TempDir()
	err := os.WriteFile(filepath.Join(dir, "roxctl-linux-amd64.tar.gz"), []byte("not a gzip"), 0600)
	s.Require().NoError(err)
	handler := handlerWithDir(dir)
	req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-linux-amd64", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	s.Equal(http.StatusInternalServerError, rr.Code)
}

func (s *HandlerSuite) TestReturns500ForCorruptedTar() {
	dir := s.T().TempDir()
	f, err := os.Create(filepath.Join(dir, "roxctl-linux-amd64.tar.gz"))
	s.Require().NoError(err)
	gz := gzip.NewWriter(f)
	_, err = gz.Write([]byte("valid gzip but not a tar"))
	s.Require().NoError(err)
	s.Require().NoError(gz.Close())
	s.Require().NoError(f.Close())
	handler := handlerWithDir(dir)
	req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-linux-amd64", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	s.Equal(http.StatusInternalServerError, rr.Code)
}
