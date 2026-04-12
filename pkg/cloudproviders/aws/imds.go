package aws

// Lightweight AWS EC2 Instance Metadata Service (IMDS) client.
// Replaces aws-sdk-go-v2/feature/ec2/imds (78 packages) with ~60 lines of net/http.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	imdsEndpoint  = "http://169.254.169.254"
	imdsTokenPath = "/latest/api/token"
	imdsTokenTTL  = "60"
	identityPath  = "/latest/dynamic/instance-identity/document"
	signedDocPath = "/latest/dynamic/instance-identity/rsa2048"
)

// InstanceIdentityDocument mirrors the fields from the EC2 instance identity document.
// This replaces imds.InstanceIdentityDocument from the AWS SDK.
type InstanceIdentityDocument struct {
	Region           string `json:"region"`
	AvailabilityZone string `json:"availabilityZone"`
	AccountID        string `json:"accountId"`
	InstanceID       string `json:"instanceId"`
	InstanceType     string `json:"instanceType"`
	ImageID          string `json:"imageId"`
}

// imdsClient is a minimal IMDS client using net/http.
type IMDSClient struct {
	httpClient *http.Client
	token      string
}

func NewIMDSClient(httpClient *http.Client) *IMDSClient {
	return &IMDSClient{httpClient: httpClient}
}

// getToken fetches an IMDSv2 session token. Falls back to IMDSv1 (no token) on failure.
func (c *IMDSClient) GetToken(ctx context.Context) {
	req, _ := http.NewRequestWithContext(ctx, "PUT", imdsEndpoint+imdsTokenPath, nil)
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", imdsTokenTTL)
	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return // Fall back to IMDSv1
	}
	defer resp.Body.Close()
	tokenBytes, _ := io.ReadAll(resp.Body)
	c.token = string(tokenBytes)
}

// get performs a GET request to the IMDS endpoint with optional IMDSv2 token.
func (c *IMDSClient) Get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imdsEndpoint+path, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("X-aws-ec2-metadata-token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDS %s returned %d", path, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// getIdentityDocument fetches and parses the instance identity document.
func (c *IMDSClient) GetIdentityDocument(ctx context.Context) (*InstanceIdentityDocument, error) {
	data, err := c.Get(ctx, identityPath)
	if err != nil {
		return nil, err
	}
	var doc InstanceIdentityDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// getSignedIdentityDocument fetches the RSA-2048 signed identity document (raw PKCS7).
func (c *IMDSClient) GetSignedIdentityDocument(ctx context.Context) ([]byte, error) {
	return c.Get(ctx, signedDocPath)
}

// getMetadata fetches a metadata value by path.
func (c *IMDSClient) GetMetadata(ctx context.Context, path string) (string, error) {
	data, err := c.Get(ctx, "/latest/meta-data/"+path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// getRegion fetches the instance region from IMDS.
func (c *IMDSClient) GetRegion(ctx context.Context) (string, error) {
	doc, err := c.GetIdentityDocument(ctx)
	if err != nil {
		return "", err
	}
	return doc.Region, nil
}

// newIMDSClientWithTimeout creates an IMDS client with the given timeout.
func NewIMDSClientWithTimeout(timeout time.Duration) *IMDSClient {
	client := &IMDSClient{
		httpClient: &http.Client{Timeout: timeout},
	}
	return client
}
