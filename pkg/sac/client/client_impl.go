package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	contentType = "application/json"
)

var (
	log = logging.LoggerForModule()
)

type clientImpl struct {
	client *http.Client
	config *storage.HTTPEndpointConfig
}

func (c *clientImpl) ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) ([]payload.AccessScope, []payload.AccessScope, error) {
	request := &payload.AuthorizationRequest{Principal: principal, RequestedScopes: scopes}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		// Only log the error message, as it might contain parts of the body that are sensitive (cluster names).
		log.Warnf("serializing: %s", err)
		return nil, nil, errors.New("could not serialize authorization plugin request")
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.config.GetEndpoint(), bytes.NewBuffer(jsonBytes))
	if err != nil {
		// This does contain the endpoint URL at worst, which is not considered sensitive.
		return nil, nil, errors.Wrap(err, "could not create HTTP request for contacting authorization plugin")
	}
	httpReq.Header.Set("content-type", contentType)
	httpReq = httpReq.WithContext(ctx)
	applyConfig(c.config, httpReq)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		// err is always a transport error, not an application error (that is stored in the response). This is not
		// sensitive.
		return nil, nil, errors.Wrap(err, "could not contact authorization plugin")
	}
	defer utils.IgnoreError(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		statusString := fmt.Sprintf("Auth plugin returned non-200 status code %s", resp.Status)
		respBytes, bodyErr := io.ReadAll(resp.Body)
		bodyOrErr := ""
		if bodyErr != nil {
			bodyOrErr = fmt.Sprintf(".  Error retrieving response body was %s", bodyErr)
		} else {
			bodyOrErr = fmt.Sprintf(".  Response body: %s", string(respBytes))
		}
		log.Warnf("%s%s", statusString, bodyOrErr)
		// Log the full error, but only return the status code, which is not sensitive.
		return nil, nil, errors.New(statusString)
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// io.ReadAll error never contains part of the read data, so it is OK to forward to the user.
		return nil, nil, errors.Wrap(err, "error reading response from authorization plugin")
	}
	var response payload.AuthorizationResponse
	if err = json.Unmarshal(respBytes, &response); err != nil {
		log.Warnf("deserializing: %s, %s", err, string(respBytes))
		// `err` might contain parts of the message, so do not forward it to the user.
		return nil, nil, errors.New("could not unmarshal authorization plugin response")
	}
	responseSet := make(map[payload.AccessScope]struct{}, len(response.AuthorizedScopes))
	for _, authScope := range response.AuthorizedScopes {
		responseSet[authScope] = struct{}{}
	}
	denied := make([]payload.AccessScope, 0, len(scopes)-len(response.AuthorizedScopes))
	for _, requestScope := range scopes {
		if _, ok := responseSet[requestScope]; !ok {
			denied = append(denied, requestScope)
		}
	}
	return response.AuthorizedScopes, denied, nil
}

func applyConfig(config *storage.HTTPEndpointConfig, req *http.Request) {
	if config.GetUsername() != "" && config.GetPassword() != "" {
		req.SetBasicAuth(config.GetUsername(), config.GetPassword())
	}
	for _, header := range config.GetHeaders() {
		if header.GetKey() != "" && header.GetValue() != "" {
			req.Header.Add(header.GetKey(), header.GetValue())
		}
	}
}
