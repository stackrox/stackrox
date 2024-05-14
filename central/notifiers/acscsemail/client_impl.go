package acscsemail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/utils"
)

const sendMsgPath = "/api/v1/email/sendMessage"

type clientImpl struct {
	loadToken  func() (string, error)
	url        string
	httpClient *http.Client
}

var _ Client = &clientImpl{}

var client *clientImpl

// ClientSingleton returns an instance of the default ACSCS email service client implementation.
// It uses HTTP to communicate to the ACSCS email service instead of SMTP, because the ACSCS email service
// acts as a direct proxy to underlying cloud email service APIs e.g. AWS SES. Using HTTP here has several benefits:
// 1. Matches the core knowledge of ACSCS team
// 2. Authentication with K8s service account tokens is easier
// 3. Reduced latency and complexity compared to SMTP, since SMTP involves a lot of back and forth messaging
// between client and server.
func ClientSingleton() Client {
	if client != nil {
		return client
	}

	url := fmt.Sprintf("%s:%s", env.ACSCSEmailURL.Setting(), sendMsgPath)

	client = &clientImpl{
		loadToken:  satoken.LoadTokenFromFile,
		url:        url,
		httpClient: http.DefaultClient,
	}

	return client
}

func (c *clientImpl) SendMessage(ctx context.Context, msg AcscsMessage) error {
	token, err := c.loadToken()
	if err != nil {
		return errors.Wrap(err, "failed to load authorization token")
	}

	msgBytes, err := json.Marshal(&msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(msgBytes))
	if err != nil {
		return errors.Wrap(err, "failed to build HTTP requests")
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send HTTP request")
	}
	defer utils.IgnoreError(res.Body.Close)

	if res.StatusCode > 300 {
		return fmt.Errorf("request to %s failed with HTTP status: %d", c.url, res.StatusCode)
	}

	return nil
}
