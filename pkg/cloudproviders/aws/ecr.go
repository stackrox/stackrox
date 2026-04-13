package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/httputil/proxy"
)

// ECRAuthToken holds the result of an ECR GetAuthorizationToken call.
type ECRAuthToken struct {
	// AuthorizationToken is base64-encoded "user:password" for docker login.
	AuthorizationToken string
	// ExpiresAt is when the token expires (typically 12 hours).
	ExpiresAt time.Time
	// ProxyEndpoint is the registry URL (e.g., "https://123456789012.dkr.ecr.us-east-1.amazonaws.com").
	ProxyEndpoint string
}

// GetECRAuthorizationToken calls the ECR GetAuthorizationToken API using
// the provided credentials and SigV4 signing. This replaces the entire
// aws-sdk-go-v2/service/ecr package (and its 78 transitive dependencies)
// with a single HTTP POST.
func GetECRAuthorizationToken(ctx context.Context, creds *Credentials, region string) (*ECRAuthToken, error) {
	endpoint := fmt.Sprintf("https://api.ecr.%s.amazonaws.com/", region)
	payload := []byte("{}")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerRegistry_V20150921.GetAuthorizationToken")

	// SigV4 sign the request
	SignV4(req, creds, region, "ecr", hashSHA256(payload))

	client := &http.Client{Timeout: 30 * time.Second, Transport: proxy.RoundTripper()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ECR GetAuthorizationToken: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("ECR GetAuthorizationToken returned %d: %s", resp.StatusCode, body)
	}

	var ecrResp struct {
		AuthorizationData []struct {
			AuthorizationToken string  `json:"authorizationToken"`
			ExpiresAt          float64 `json:"expiresAt"`
			ProxyEndpoint      string  `json:"proxyEndpoint"`
		} `json:"authorizationData"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ecrResp); err != nil {
		return nil, fmt.Errorf("decoding ECR response: %w", err)
	}

	if len(ecrResp.AuthorizationData) == 0 {
		return nil, fmt.Errorf("ECR returned empty authorization data")
	}

	ad := ecrResp.AuthorizationData[0]
	return &ECRAuthToken{
		AuthorizationToken: ad.AuthorizationToken,
		ExpiresAt:          time.Unix(int64(ad.ExpiresAt), 0),
		ProxyEndpoint:      ad.ProxyEndpoint,
	}, nil
}
