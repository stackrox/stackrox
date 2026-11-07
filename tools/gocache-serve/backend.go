package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// newBackend creates a cloud storage backend based on GOCACHE_SERVE_BACKEND
func newBackend() (Backend, error) {
	backendType := os.Getenv("GOCACHE_SERVE_BACKEND")
	if backendType == "" {
		backendType = "azure"
	}

	switch backendType {
	case "azure":
		return newAzureBackend()
	case "gcs":
		return newGCSBackend()
	case "s3":
		return newS3Backend()
	case "noop":
		return &NoopBackend{}, nil
	default:
		return nil, fmt.Errorf("unknown backend: %s", backendType)
	}
}

// NoopBackend does nothing (for testing the protocol without cloud)
type NoopBackend struct{}

func (n *NoopBackend) Get(actionID string) ([]byte, error) { return nil, nil }
func (n *NoopBackend) Put(actionID string, body []byte) error { return nil }
func (n *NoopBackend) Close() error { return nil }

// AzureBackend uses Azure Blob Storage via the REST API (no SDK dependency
// for the prototype — uses the same SPN auth as the FUSE experiments)
type AzureBackend struct {
	account   string
	container string
	token     string
	tokenExp  time.Time
	client    *http.Client
}

func newAzureBackend() (*AzureBackend, error) {
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if account == "" {
		return nil, fmt.Errorf("AZURE_STORAGE_ACCOUNT not set")
	}
	return &AzureBackend{
		account:   account,
		container: "ci-gocache",
		client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (a *AzureBackend) baseURL() string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", a.account, a.container)
}

func (a *AzureBackend) Get(actionID string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", a.baseURL(), actionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err := a.setAuth(req); err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure GET %s: HTTP %d", actionID[:8], resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (a *AzureBackend) Put(actionID string, body []byte) error {
	url := fmt.Sprintf("%s/%s", a.baseURL(), actionID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	if err := a.setAuth(req); err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("azure PUT %s: HTTP %d", actionID[:8], resp.StatusCode)
	}
	return nil
}

func (a *AzureBackend) Close() error { return nil }

// setAuth sets the Authorization header using Azure AD token (SPN auth)
// For the prototype, we use a shared key if available, or SPN via env
func (a *AzureBackend) setAuth(req *http.Request) error {
	// For prototype simplicity, use SAS token or public container
	// In production, this would use Azure AD token from SPN
	// The container ci-gocache is set to public read for now
	// TODO: implement proper SPN auth
	return nil
}

// GCSBackend uses Google Cloud Storage via the REST API
type GCSBackend struct {
	bucket string
	token  string
	client *http.Client
}

func newGCSBackend() (*GCSBackend, error) {
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		bucket = "ci-gocache-proto"
	}
	return &GCSBackend{
		bucket: bucket,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (g *GCSBackend) Get(actionID string) ([]byte, error) {
	url := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media", g.bucket, actionID)
	resp, err := g.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gcs GET %s: HTTP %d", actionID[:8], resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (g *GCSBackend) Put(actionID string, body []byte) error {
	url := fmt.Sprintf("https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s", g.bucket, actionID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gcs PUT %s: HTTP %d", actionID[:8], resp.StatusCode)
	}
	return nil
}

func (g *GCSBackend) Close() error { return nil }

// S3Backend uses AWS S3 via the REST API
type S3Backend struct {
	bucket string
	client *http.Client
}

func newS3Backend() (*S3Backend, error) {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "ci-gocache-proto"
	}
	return &S3Backend{
		bucket: bucket,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *S3Backend) Get(actionID string) ([]byte, error) {
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, actionID)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("s3 GET %s: HTTP %d", actionID[:8], resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (s *S3Backend) Put(actionID string, body []byte) error {
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, actionID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("s3 PUT %s: HTTP %d", actionID[:8], resp.StatusCode)
	}
	return nil
}

func (s *S3Backend) Close() error { return nil }

// helper for actionID truncation
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ = strings.TrimSpace // keep import for future use
var _ = truncate          // keep for future use
