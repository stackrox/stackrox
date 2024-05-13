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
