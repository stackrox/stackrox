package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/httputil/proxy"
)

// Credentials holds temporary AWS credentials for signing requests.
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// IsExpired returns true if the credentials have expired or will expire within the given margin.
func (c *Credentials) IsExpired(margin time.Duration) bool {
	if c.Expiration.IsZero() {
		return false // static credentials don't expire
	}
	return time.Now().Add(margin).After(c.Expiration)
}

// ResolveCredentials tries the standard AWS credential chain:
// 1. Environment variables
// 2. IRSA (IAM Roles for Service Accounts) — web identity token → STS
// 3. IMDS (EC2 instance metadata)
func ResolveCredentials(ctx context.Context, region string) (*Credentials, error) {
	// 1. Environment variables
	if key := os.Getenv("AWS_ACCESS_KEY_ID"); key != "" {
		return &Credentials{
			AccessKeyID:     key,
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		}, nil
	}

	// 2. IRSA — web identity token file
	tokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	roleARN := os.Getenv("AWS_ROLE_ARN")
	if tokenFile != "" && roleARN != "" {
		return assumeRoleWithWebIdentity(ctx, region, tokenFile, roleARN)
	}

	// 3. IMDS — EC2 instance role
	return credentialsFromIMDS(ctx)
}

// assumeRoleWithWebIdentity calls STS AssumeRoleWithWebIdentity.
// This is a public API that does NOT require SigV4 signing.
func assumeRoleWithWebIdentity(ctx context.Context, region, tokenFile, roleARN string) (*Credentials, error) {
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("reading web identity token: %w", err)
	}

	stsEndpoint := "https://sts.amazonaws.com/"
	if region != "" {
		stsEndpoint = fmt.Sprintf("https://sts.%s.amazonaws.com/", region)
	}

	form := url.Values{
		"Action":           {"AssumeRoleWithWebIdentity"},
		"Version":          {"2011-06-15"},
		"RoleArn":          {roleARN},
		"RoleSessionName":  {"stackrox-sensor"},
		"WebIdentityToken": {strings.TrimSpace(string(token))},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", stsEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second, Transport: proxy.RoundTripper()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("STS AssumeRoleWithWebIdentity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("STS AssumeRoleWithWebIdentity returned %d: %s", resp.StatusCode, body)
	}

	return parseSTSResponse(resp.Body)
}

// parseSTSResponse extracts credentials from the STS XML response.
func parseSTSResponse(r io.Reader) (*Credentials, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// Parse the XML response with simple string extraction.
	// The response format is well-defined and stable.
	extract := func(tag string) string {
		start := strings.Index(string(body), "<"+tag+">")
		end := strings.Index(string(body), "</"+tag+">")
		if start < 0 || end < 0 {
			return ""
		}
		return string(body[start+len(tag)+2 : end])
	}

	accessKey := extract("AccessKeyId")
	secretKey := extract("SecretAccessKey")
	sessionToken := extract("SessionToken")
	expirationStr := extract("Expiration")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("STS response missing credentials: %s", body)
	}

	var expiration time.Time
	if expirationStr != "" {
		expiration, _ = time.Parse(time.RFC3339, expirationStr)
	}

	return &Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		Expiration:      expiration,
	}, nil
}

// credentialsFromIMDS fetches IAM role credentials from the EC2 instance metadata service.
func credentialsFromIMDS(ctx context.Context) (*Credentials, error) {
	client := NewIMDSClient(&http.Client{
		Timeout:   30 * time.Second,
		Transport: proxy.Without(),
	})
	client.GetToken(ctx)

	// Get the IAM role name
	roleName, err := client.GetMetadata(ctx, "iam/security-credentials/")
	if err != nil {
		return nil, fmt.Errorf("getting IAM role from IMDS: %w", err)
	}
	roleName = strings.TrimSpace(roleName)

	// Get credentials for the role
	credsJSON, err := client.GetMetadata(ctx, "iam/security-credentials/"+roleName)
	if err != nil {
		return nil, fmt.Errorf("getting credentials for role %q from IMDS: %w", roleName, err)
	}

	var imdsResp struct {
		AccessKeyId     string    `json:"AccessKeyId"`
		SecretAccessKey string    `json:"SecretAccessKey"`
		Token           string    `json:"Token"`
		Expiration      time.Time `json:"Expiration"`
	}
	if err := json.Unmarshal([]byte(credsJSON), &imdsResp); err != nil {
		return nil, fmt.Errorf("parsing IMDS credentials: %w", err)
	}

	return &Credentials{
		AccessKeyID:     imdsResp.AccessKeyId,
		SecretAccessKey: imdsResp.SecretAccessKey,
		SessionToken:    imdsResp.Token,
		Expiration:      imdsResp.Expiration,
	}, nil
}
