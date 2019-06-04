package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	contentType = "application/json"
	log         = logging.LoggerForModule()
)

type clientImpl struct {
	client       *http.Client
	authEndpoint string
}

func (c *clientImpl) ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) ([]payload.AccessScope, []payload.AccessScope, error) {
	request := &payload.AuthorizationRequest{Principal: principal, RequestedScopes: scopes}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.authEndpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("content-type", contentType)
	httpReq = httpReq.WithContext(ctx)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer utils.IgnoreError(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		errString := fmt.Sprintf("Auth plugin returned non-200 status code %s", resp.Status)
		respBytes, bodyErr := ioutil.ReadAll(resp.Body)
		bodyOrErr := ""
		if bodyErr != nil {
			bodyOrErr = fmt.Sprintf(".  Error retrieving response body was %s", bodyErr.Error())
		} else {
			bodyOrErr = fmt.Sprintf(".  Response body: %s", string(respBytes))
		}
		log.Warnf("%s%s", errString, bodyOrErr)
		return nil, nil, errors.New(errString)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var response payload.AuthorizationResponse
	if err = json.Unmarshal(respBytes, &response); err != nil {
		return nil, nil, err
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
