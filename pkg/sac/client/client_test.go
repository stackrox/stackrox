package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/suite"
)

var (
	allowEndpoint = "/authRequestSuccess"
	denyEndpoint  = "/authRequestFailure"
	errorEndpoint = "/authRequestError"
	scopes        = []payload.AccessScope{
		{
			Verb: "Verb",
			Noun: "Noun",
			Attributes: payload.NounAttributes{
				Namespace: "Namespace",
				Cluster: payload.Cluster{
					ID: "ID", Name: "Name",
				},
			},
		},
	}
)

func TestClient(t *testing.T) {
	suite.Run(t, new(clientTestSuite))
}

type clientTestSuite struct {
	suite.Suite

	server *httptest.Server
}

func (suite *clientTestSuite) SetupTest() {
	router := http.NewServeMux()
	router.HandleFunc(allowEndpoint, suite.allowAll)
	router.HandleFunc(denyEndpoint, suite.denyAll)
	router.HandleFunc(errorEndpoint, suite.errorResponse)
	suite.server = httptest.NewServer(router)
}

func (suite *clientTestSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *clientTestSuite) getTestClient(endpoint string) *clientImpl {
	httpClient := suite.server.Client()
	config := &storage.HTTPEndpointConfig{Endpoint: "http://" + suite.server.Listener.Addr().String() + endpoint}
	return &clientImpl{client: httpClient, config: config}
}

func (suite *clientTestSuite) allowAll(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	request := new(payload.AuthorizationRequest)
	reqBytes, err := io.ReadAll(r.Body)
	suite.NoError(err)
	err = json.Unmarshal(reqBytes, request)
	suite.NoError(err)
	response := &payload.AuthorizationResponse{AuthorizedScopes: request.RequestedScopes}
	err = json.NewEncoder(w).Encode(response)
	suite.NoError(err)
}

func (suite *clientTestSuite) denyAll(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(&payload.AuthorizationResponse{})
	suite.NoError(err)
}

func (suite *clientTestSuite) errorResponse(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusTeapot)
}

func (suite *clientTestSuite) TestAllow() {
	client := suite.getTestClient(allowEndpoint)
	allowed, denied, err := client.ForUser(context.Background(), payload.Principal{}, scopes...)
	suite.NoError(err)
	suite.ElementsMatch(scopes, allowed)
	suite.Empty(denied)
}

func (suite *clientTestSuite) TestDeny() {
	client := suite.getTestClient(denyEndpoint)
	allowed, denied, err := client.ForUser(context.Background(), payload.Principal{}, scopes...)
	suite.NoError(err)
	suite.Empty(allowed)
	suite.ElementsMatch(scopes, denied)
}

func (suite *clientTestSuite) TestError() {
	client := suite.getTestClient(errorEndpoint)
	allowed, denied, err := client.ForUser(context.Background(), payload.Principal{}, scopes...)
	suite.Error(err)
	suite.Nil(allowed)
	suite.Nil(denied)
}
