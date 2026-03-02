package datastore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
)

func TestUpdater(t *testing.T) {
	suite.Run(t, new(updaterTestSuite))
}

type updaterTestSuite struct {
	suite.Suite
}

// validPEM is a valid PUBLIC KEY PEM for tests (from validate_test.go goodCosignConfig).
func (s *updaterTestSuite) validPEM() string {
	return goodCosignConfig.GetPublicKeys()[0].GetPublicKeyPemEnc()
}

func (s *updaterTestSuite) TestResolveKeyURL() {
	tests := []struct {
		manifestURL string
		keyURL      string
		want        string
		wantErr     bool
	}{
		{"https://a.com/dir/manifest.json", "https://b.com/key.pub", "https://b.com/key.pub", false},
		{"https://a.com/dir/manifest.json", "http://b.com/key.pub", "http://b.com/key.pub", false},
		{"https://example.com/keys/manifest.json", "release-key.pub", "https://example.com/keys/release-key.pub", false},
		{"https://example.com/keys/manifest.json", "sub/key.pub", "https://example.com/keys/sub/key.pub", false},
		{"https://example.com/manifest.json", "key.pub", "https://example.com/key.pub", false},
		{"https://example.com/keys/manifest.json", "https://other.com/key.pub", "https://other.com/key.pub", false},
		{"https://example.com/keys/manifest.json", "https://example.com/keys/", "", true}, // key URL is a directory
		{"https://example.com/keys/manifest.json", "keys/", "", true},                     // key URL is a directory
		{"://invalid", "key.pub", "", true},
	}
	for _, tt := range tests {
		got, err := resolveKeyURL(tt.manifestURL, tt.keyURL)
		if tt.wantErr {
			s.Require().Error(err, "manifestURL=%q keyURL=%q", tt.manifestURL, tt.keyURL)
			continue
		}
		s.Require().NoError(err, "manifestURL=%q keyURL=%q", tt.manifestURL, tt.keyURL)
		s.Equal(tt.want, got, "manifestURL=%q keyURL=%q", tt.manifestURL, tt.keyURL)
	}
}

func (s *updaterTestSuite) TestValidatePublicKey() {
	s.NoError(validatePublicKey(s.validPEM()))
	s.Error(validatePublicKey("not-pem"))
	s.Error(validatePublicKey("-----BEGIN PUBLIC KEY-----\n!!!\n-----END PUBLIC KEY-----"))
	s.Error(validatePublicKey(""))
}

func (s *updaterTestSuite) TestFetchManifest() {
	validManifest := publicKeyManifest{
		Keys: []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}{
			{Name: "Key1", URL: "key1.pub"},
			{Name: "Key2", URL: "key2.pub"},
		},
	}
	body, _ := json.Marshal(validManifest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	u := s.newTestUpdater(server.URL+"/manifest.json", time.Second)

	// Success
	m, err := u.fetchManifest(server.URL + "/manifest.json")
	s.NoError(err)
	s.Len(m.Keys, 2)
	s.Equal("Key1", m.Keys[0].Name)
	s.Equal("key1.pub", m.Keys[0].URL)

	// Non-200
	_, err = u.fetchManifest(server.URL + "/nonexistent")
	s.Error(err)
	s.Contains(err.Error(), "404")

	// Invalid JSON
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer badServer.Close()
	_, err = u.fetchManifest(badServer.URL)
	s.Error(err)
	s.Contains(err.Error(), "unmarshalling")
}

func (s *updaterTestSuite) TestFetchPublicKey() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(s.validPEM()))
	}))
	defer server.Close()

	u := s.newTestUpdater(server.URL, time.Second)

	key, err := u.fetchPublicKey("MyKey", server.URL)
	s.NoError(err)
	s.Equal("MyKey", key.name)
	s.Equal(s.validPEM(), key.publicKeyPemEnc)

	// Non-200
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()
	_, err = u.fetchPublicKey("MyKey", badServer.URL)
	s.Error(err)
	s.Contains(err.Error(), "500")

	// Invalid PEM
	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-pem"))
	}))
	defer invalidServer.Close()
	_, err = u.fetchPublicKey("MyKey", invalidServer.URL)
	s.Error(err)
	s.Contains(err.Error(), "validating")
}

func (s *updaterTestSuite) TestFetchPublicKeysFromManifest() {
	validPEM := s.validPEM()
	manifest := publicKeyManifest{
		Keys: []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}{
			{Name: "Key A", URL: "/key-a.pub"},
			{Name: "Key B", URL: "/key-b.pub"},
		},
	}
	manifestBody, _ := json.Marshal(manifest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
		case "/key-a.pub", "/key-b.pub":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validPEM))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	u := s.newTestUpdater(server.URL+"/manifest.json", time.Second)

	keys, err := u.fetchPublicKeysFromManifest(server.URL + "/manifest.json")
	s.NoError(err)
	s.Len(keys, 2)
	s.Equal("Key A", keys[0].name)
	s.Equal("Key B", keys[1].name)

	// One key 404: that key is skipped
	serverPartial := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
		} else if r.URL.Path == "/key-a.pub" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validPEM))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer serverPartial.Close()
	u2 := s.newTestUpdater(serverPartial.URL+"/manifest.json", time.Second)
	keys2, err := u2.fetchPublicKeysFromManifest(serverPartial.URL + "/manifest.json")
	s.NoError(err)
	s.Len(keys2, 1)
	s.Equal("Key A", keys2[0].name)

	// All keys fail: empty slice
	serverAllFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer serverAllFail.Close()
	u3 := s.newTestUpdater(serverAllFail.URL+"/manifest.json", time.Second)
	keys3, err := u3.fetchPublicKeysFromManifest(serverAllFail.URL + "/manifest.json")
	s.NoError(err)
	s.Empty(keys3)
}

// newTestUpdater builds an updater with a custom manifest URL for testing.
func (s *updaterTestSuite) newTestUpdater(manifestURL string, interval time.Duration) *updater {
	return &updater{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		interval:    interval,
		manifestURL: manifestURL,
		stopSig:     concurrency.NewSignal(),
	}
}
