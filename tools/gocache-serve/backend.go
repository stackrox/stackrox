package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Backend interface for cloud storage
type Backend interface {
	Get(actionID string) ([]byte, error) // nil, nil = not found
	Put(actionID string, body []byte) error
	PutIfNotExist(actionID string, body []byte) (bool, error) // atomic; false = exists
	ListKeys() ([]string, error)                              // list all blob keys in the container
	Close() error
}

// NoopBackend does nothing (for protocol testing)
type NoopBackend struct{}

func (n *NoopBackend) Get(string) ([]byte, error)                 { return nil, nil }
func (n *NoopBackend) Put(string, []byte) error                   { return nil }
func (n *NoopBackend) PutIfNotExist(string, []byte) (bool, error) { return true, nil }
func (n *NoopBackend) ListKeys() ([]string, error)                { return nil, nil }
func (n *NoopBackend) Close() error                               { return nil }

// AzureBackend talks directly to Azure Blob Storage via REST API.
// Zero dependencies — uses net/http with SPN OAuth for auth.
type AzureBackend struct {
	account      string
	container    string
	tenantID     string
	clientID     string
	clientSecret string

	tokenMu  sync.Mutex
	token    string
	tokenExp time.Time

	client *http.Client
}

// detectRegion returns the runner's Azure region via Instance Metadata
// Service (IMDS). Falls back to the default region if IMDS is unavailable.
func detectRegion() string {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET",
		"http://169.254.169.254/metadata/instance/compute/location?api-version=2021-02-01&format=text", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata", "true")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// storageAccountForRegion selects the nearest storage account for a runner region.
// The primary is westus3; eastus is an object-replication replica.
func storageAccountForRegion(region string) string {
	switch region {
	case "eastus", "eastus2", "centralus", "northcentralus", "southcentralus",
		"westeurope", "northeurope", "uk south", "francecentral":
		return "cigocacheeast" // eastus replica
	case "westus3", "westus2", "westus", "northwestus",
		"eastasia", "southeastasia", "japaneast", "japanwest",
		"australiaeast", "australiasoutheast":
		return "cigocache" // westus3 primary
	default:
		return "cigocache" // fallback to primary
	}
}

// containerFromEnv returns the blob container name, overridable for
// A/B experiments (fresh containers = identical cold-start state).
func containerFromEnv() string {
	if c := os.Getenv("GOCACHE_SERVE_CONTAINER"); c != "" {
		return c
	}
	return "ci-gocache"
}

func newAzureBackend() (*AzureBackend, error) {
	// Use explicitly-set account, or detect from runner region
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if account == "" {
		region := detectRegion()
		account = storageAccountForRegion(region)
		log.Printf("gocache-serve: region=%s account=%s", region, account)
	}
	if account == "" {
		return nil, fmt.Errorf("AZURE_STORAGE_ACCOUNT not set and region detection failed")
	}
	b := &AzureBackend{
		account:      account,
		container:    containerFromEnv(),
		tenantID:     os.Getenv("AZURE_TENANT_ID"),
		clientID:     os.Getenv("AZURE_CLIENT_ID"),
		clientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		client:       &http.Client{Timeout: 30 * time.Second},
	}
	if b.tenantID == "" || b.clientID == "" || b.clientSecret == "" {
		return nil, fmt.Errorf("Azure SPN env not set (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)")
	}
	return b, nil
}

func (a *AzureBackend) containerURL() string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", a.account, a.container)
}

func (a *AzureBackend) blobURL(actionID string) string {
	return fmt.Sprintf("%s/%s", a.containerURL(), actionID)
}

// getToken fetches an OAuth token via client credentials, cached until expiry.
func (a *AzureBackend) getToken() (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.token != "" && time.Now().Before(a.tokenExp.Add(-time.Minute)) {
		return a.token, nil
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.tenantID)
	form := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s&scope=https://storage.azure.com/.default",
		a.clientID, a.clientSecret,
	)
	resp, err := a.client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("token decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty token")
	}

	a.token = tr.AccessToken
	a.tokenExp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return a.token, nil
}

func (a *AzureBackend) Get(actionID string) ([]byte, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", a.blobURL(actionID), nil)
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
		return nil, nil // miss
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", shortID(actionID), resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (a *AzureBackend) Put(actionID string, body []byte) error {
	token, err := a.getToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", a.blobURL(actionID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("PUT %s: HTTP %d", shortID(actionID), resp.StatusCode)
	}
	return nil
}

// PutIfNotExist writes a blob only if it does not already exist (atomic).
// Returns false if another writer got there first (HTTP 409/412).
// Used for advisory compilation leases between concurrent runners.
func (a *AzureBackend) PutIfNotExist(actionID string, body []byte) (bool, error) {
	token, err := a.getToken()
	if err != nil {
		return false, err
	}
	req, err := http.NewRequest("PUT", a.blobURL(actionID), bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
	req.Header.Set("If-None-Match", "*") // atomic: fail if blob exists

	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return true, nil
	}
	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusPreconditionFailed {
		return false, nil // someone else holds it
	}
	return false, fmt.Errorf("PUT-if-none-match %s: HTTP %d", shortID(actionID), resp.StatusCode)
}

// DeleteKey is intentionally NOT used for lease release: deleting the lease
// right after an entry upload would let late missers (whose GET raced the
// upload) acquire and recompile redundantly instead of waiting-and-hitting.
// Lease blobs are GC'd server-side by a lifecycle rule (see lease.go).

// ListKeys returns all blob names in the container (paginated).
func (a *AzureBackend) ListKeys() ([]string, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, err
	}

	var keys []string
	marker := ""
	for {
		url := a.containerURL() + "?restype=container&comp=list&maxresults=5000"
		if marker != "" {
			url += "&marker=" + marker
		}
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
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("list: HTTP %d", resp.StatusCode)
		}

		var listResp struct {
			Blobs []struct {
				Name string `xml:"Name"`
			} `xml:"Blobs>Blob"`
			NextMarker string `xml:"NextMarker"`
		}
		if err := xml.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			return nil, fmt.Errorf("list decode: %w", err)
		}
		for _, b := range listResp.Blobs {
			keys = append(keys, b.Name)
		}
		if listResp.NextMarker == "" {
			break
		}
		marker = listResp.NextMarker
	}
	return keys, nil
}

func (a *AzureBackend) Close() error { return nil }

func shortID(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func newBackend() (Backend, error) {
	switch os.Getenv("GOCACHE_SERVE_BACKEND") {
	case "", "azure":
		return newAzureBackend()
	case "noop":
		return &NoopBackend{}, nil
	default:
		return nil, fmt.Errorf("unknown backend (azure, noop)")
	}
}
