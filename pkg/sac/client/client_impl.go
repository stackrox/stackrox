package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	contentType = "application/json"
)

var (
	log                   = logging.LoggerForModule()
	errAuthzPluginContact = errors.New("contacting auth server")
)

type clientImpl struct {
	client *http.Client
	config *storage.HTTPEndpointConfig
}

func (c *clientImpl) ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) ([]payload.AccessScope, []payload.AccessScope, error) {
	request := &payload.AuthorizationRequest{Principal: principal, RequestedScopes: scopes}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		log.Warnf("serializing: %s", err)
		return nil, nil, errAuthzPluginContact
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.config.GetEndpoint(), bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Warnf("creating: %s, %s", err, string(jsonBytes))
		return nil, nil, errAuthzPluginContact
	}
	httpReq.Header.Set("content-type", contentType)
	httpReq = httpReq.WithContext(ctx)
	applyConfig(c.config, httpReq)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		log.Warnf("sending: %s", err)
		return nil, nil, errAuthzPluginContact
	}
	defer utils.IgnoreError(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		statusString := fmt.Sprintf("Auth plugin returned non-200 status code %s", resp.Status)
		respBytes, bodyErr := ioutil.ReadAll(resp.Body)
		bodyOrErr := ""
		if bodyErr != nil {
			bodyOrErr = fmt.Sprintf(".  Error retrieving response body was %s", bodyErr)
		} else {
			bodyOrErr = fmt.Sprintf(".  Response body: %s", string(respBytes))
		}
		log.Warnf("%s%s", statusString, bodyOrErr)
		return nil, nil, errAuthzPluginContact
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("reading: %s", err)
		return nil, nil, errAuthzPluginContact
	}
	var response payload.AuthorizationResponse
	if err = json.Unmarshal(respBytes, &response); err != nil {
		log.Warnf("deserializing: %s, %s", err, string(respBytes))
		return nil, nil, errAuthzPluginContact
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
