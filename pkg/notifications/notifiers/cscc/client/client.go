package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/notifications/notifiers/cscc/findings"
	"golang.org/x/oauth2/google"
)

const (
	cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
	timeout            = 5 * time.Second
)

// Logger is the minimal interface we need to use to log data.
type Logger interface {
	Warnf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

// A Config contains the necessary information to make a CSCC request.
type Config struct {
	GCPOrganizationID string
	ServiceAccount    []byte
	Logger            Logger
}

func (c *Config) url() string {
	return fmt.Sprintf(
		"https://securitycenter.googleapis.com/v1alpha3/organizations/%s/findings",
		c.GCPOrganizationID,
	)
}

// CreateFinding creates the provided SourceFinding.
func (c *Config) CreateFinding(finding *findings.SourceFinding) error {
	req, err := c.request(finding)
	if err != nil {
		return fmt.Errorf("request creation: %s", err)
	}

	ctx, cancel := timeoutContext()
	defer cancel()
	tokenSource, err := c.getTokenSource(ctx)
	if err != nil {
		return fmt.Errorf("token source retrieval: %s", err)
	}

	token, err := tokenSource.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("token retrieval: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %s", err)
	}
	defer resp.Body.Close()
	return c.handleResponse(resp)
}

func (c *Config) request(finding *findings.SourceFinding) (*http.Request, error) {
	b, err := json.Marshal(&findings.CreateFindingMessage{
		Finding: *finding,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %s", err)
	}

	req, err := http.NewRequest("POST", c.url(), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("build: %s", err)
	}
	return req, nil
}

func (c *Config) handleResponse(r *http.Response) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.Logger.Warnf("Response decoding failed: %s", err)
	}
	c.Logger.Debugf("CSCC response: %d %s; %s", r.StatusCode, r.Status, string(b))
	if r.StatusCode >= 400 {
		return fmt.Errorf("Unexpected response code %d: %s", r.StatusCode, string(b))
	}
	return nil
}

func (c *Config) getTokenSource(ctx context.Context) (*google.DefaultCredentials, error) {
	cfg, err := google.JWTConfigFromJSON(c.ServiceAccount, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("google.JWTConfigFromJSON: %s", err)
	}
	pid, err := c.embeddedProjectID()
	if err != nil {
		return nil, fmt.Errorf("project ID retrieval: %s", err)
	}
	return &google.DefaultCredentials{
		ProjectID:   pid,
		TokenSource: cfg.TokenSource(ctx),
		JSON:        c.ServiceAccount,
	}, nil
}

func timeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func (c *Config) embeddedProjectID() (string, error) {
	// jwt.Config does not expose the project ID, so re-unmarshal to get it.
	var pid struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(c.ServiceAccount, &pid); err != nil {
		return "", err
	}
	return pid.ProjectID, nil
}
