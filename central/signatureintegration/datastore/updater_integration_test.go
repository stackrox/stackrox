//go:build sql_integration

package datastore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/suite"
)

const validTestPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO
3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX
7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS
j+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd
OrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ
5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl
FQIDAQAB
-----END PUBLIC KEY-----`

func TestUpdaterIntegration(t *testing.T) {
	suite.Run(t, new(updaterIntegrationTestSuite))
}

type updaterIntegrationTestSuite struct {
	suite.Suite

	ctx     context.Context
	db      *pgtest.TestPostgres
	storage postgres.Store
}

func (s *updaterIntegrationTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))

	s.db = pgtest.ForT(s.T())
	s.storage = postgres.New(s.db)
	siStore = s.storage
}

func (s *updaterIntegrationTestSuite) verifyStoredIntegration(expected *storage.SignatureIntegration) {
	s.T().Helper()
	stored, exists, err := s.storage.Get(s.ctx, expected.GetId())
	s.Require().NoError(err)
	s.Require().True(exists)
	protoassert.Equal(s.T(), expected, stored)
}

func (s *updaterIntegrationTestSuite) verifyStoredKeys(integrationID string, expectedKeys []*storage.CosignPublicKeyVerification_PublicKey) {
	s.T().Helper()
	stored, exists, err := s.storage.Get(s.ctx, integrationID)
	s.Require().NoError(err)
	s.Require().True(exists)
	actual := stored.GetCosign().GetPublicKeys()
	s.Require().Len(actual, len(expectedKeys))
	for i := range expectedKeys {
		s.Equal(expectedKeys[i].GetName(), actual[i].GetName())
		s.Equal(expectedKeys[i].GetPublicKeyPemEnc(), actual[i].GetPublicKeyPemEnc())
	}
}

func (s *updaterIntegrationTestSuite) newTestUpdater(manifestURL string) *updater {
	return &updater{
		client:      &http.Client{Timeout: 5 * time.Second},
		interval:    time.Second,
		manifestURL: manifestURL,
		stopSig:     concurrency.NewSignal(),
	}
}

func (s *updaterIntegrationTestSuite) TestStoredIntegrationUnchangedOnFailure() {
	original := signatures.DefaultRedHatSignatureIntegration.CloneVT()
	s.Require().NoError(upsertDefaultRedHatSignatureIntegration(s.storage, original))
	s.verifyStoredIntegration(original)

	s.Run("manifest fetch fails", func() {
		u := s.newTestUpdater("http://localhost:0/manifest.json")
		err := u.update()
		s.Error(err)
		s.verifyStoredIntegration(original)
	})

	s.Run("all key fetches fail", func() {
		manifestBody, _ := json.Marshal(publicKeyManifest{
			Keys: []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			}{
				{Name: "K", URL: "/key.pub"},
			},
		})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/manifest.json" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(manifestBody)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		u := s.newTestUpdater(server.URL + "/manifest.json")
		err := u.update()
		s.Error(err)
		s.Contains(err.Error(), "no valid public keys")
		s.verifyStoredIntegration(original)
	})

	s.Run("key validation fails", func() {
		manifestBody, _ := json.Marshal(publicKeyManifest{
			Keys: []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			}{
				{Name: "K", URL: "/key.pub"},
			},
		})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/manifest.json" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(manifestBody)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid-pem"))
			}
		}))
		defer server.Close()

		u := s.newTestUpdater(server.URL + "/manifest.json")
		err := u.update()
		s.Error(err)
		s.Contains(err.Error(), "no valid public keys")
		s.verifyStoredIntegration(original)
	})
}

func (s *updaterIntegrationTestSuite) TestStoredIntegrationUpdatedOnSuccess() {
	original := signatures.DefaultRedHatSignatureIntegration.CloneVT()
	s.Require().NoError(upsertDefaultRedHatSignatureIntegration(s.storage, original))
	s.verifyStoredIntegration(original)

	// One key: success
	s.Run("single key", func() {
		manifestBody, _ := json.Marshal(publicKeyManifest{
			Keys: []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			}{
				{Name: "Red Hat Release Key 3", URL: "/key.pub"},
			},
		})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/manifest.json" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(manifestBody)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(validTestPublicKey))
			}
		}))
		defer server.Close()

		u := s.newTestUpdater(server.URL + "/manifest.json")
		s.NoError(u.update())
		s.verifyStoredKeys(original.GetId(), []*storage.CosignPublicKeyVerification_PublicKey{
			{Name: "Red Hat Release Key 3", PublicKeyPemEnc: validTestPublicKey},
		})
	})

	// Two keys: both stored
	s.Run("multiple keys", func() {
		manifestBody, _ := json.Marshal(publicKeyManifest{
			Keys: []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			}{
				{Name: "Key A", URL: "/a.pub"},
				{Name: "Key B", URL: "/b.pub"},
			},
		})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/manifest.json" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(manifestBody)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(validTestPublicKey))
			}
		}))
		defer server.Close()

		u := s.newTestUpdater(server.URL + "/manifest.json")
		s.NoError(u.update())
		s.verifyStoredKeys(original.GetId(), []*storage.CosignPublicKeyVerification_PublicKey{
			{Name: "Key A", PublicKeyPemEnc: validTestPublicKey},
			{Name: "Key B", PublicKeyPemEnc: validTestPublicKey},
		})
	})

	// One key 404, one valid: one stored
	s.Run("partial key failure", func() {
		manifestBody, _ := json.Marshal(publicKeyManifest{
			Keys: []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			}{
				{Name: "Good", URL: "/good.pub"},
				{Name: "Bad", URL: "/bad.pub"},
			},
		})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/manifest.json" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(manifestBody)
			} else if r.URL.Path == "/good.pub" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(validTestPublicKey))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		u := s.newTestUpdater(server.URL + "/manifest.json")
		s.NoError(u.update())
		s.verifyStoredKeys(original.GetId(), []*storage.CosignPublicKeyVerification_PublicKey{
			{Name: "Good", PublicKeyPemEnc: validTestPublicKey},
		})
	})
}
