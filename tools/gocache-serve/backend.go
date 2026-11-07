package main

import (
	"bytes"
	"encoding/json"
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

// AzureBackend uses Azure Blob Storage via the REST API with SPN auth
type AzureBackend struct {
	account   string
	container string
	tenantID  string
	clientID  string
	clientSecret string
	token     string
	tokenExp  time.Time
	client    *http.Client
}

func newAzureBackend() (*AzureBackend, error) {
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if account == "" {
		return nil, fmt.Errorf("AZURE_STORAGE_ACCOUNT not set")
	}
	b := &AzureBackend{
		account:      account,
		container:    "ci-gocache",
		tenantID:     os.Getenv("AZURE_TENANT_ID"),
		clientID:     os.Getenv("AZURE_CLIENT_ID"),
		clientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		client:       &http.Client{Timeout: 30 * time.Second},
	}
	if b.tenantID == "" || b.clientID == "" || b.clientSecret == "" {
		return nil, fmt.Errorf("Azure SPN credentials not set (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)")
	}
	return b, nil
}

func (a *AzureBackend) baseURL() string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", a.account, a.container)
}

// getToken obtains an OAuth token via the client credentials flow
func (a *AzureBackend) getToken() (string, error) {
	if a.token != "" && time.Now().Before(a.tokenExp.Add(-time.Minute)) {
		return a.token, nil
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.tenantID)
	data := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s&scope=%s",
		a.clientID, a.clientSecret,
		"https://storage.azure.com/.default",
	)
	resp, err := a.client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}

	a.token = tokenResp.AccessToken
	a.tokenExp = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return a.token, nil
}

func (a *AzureBackend) Get(actionID string) ([]byte, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	url := fmt.Sprintf("%s/%s", a.baseURL(), actionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-version", "2023-11-03")
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
	token, err := a.getToken()
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	url := fmt.Sprintf("%s/%s", a.baseURL(), actionID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
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
