package httputil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func endpointReturningProto(_ *http.Request) (*storage.Alert, error) {
	return fixtures.GetSerializationTestAlert(), nil
}

type jsonMarshalable struct {
	Name string `json:"name"`
}

func endpointReturningJSONMarshalable(_ *http.Request) (*jsonMarshalable, error) {
	return &jsonMarshalable{
		Name: "jsonMarshalable",
	}, nil
}

func TestRESTHandler(t *testing.T) {
	for testName, testCase := range map[string]struct {
		endpointFunc     func(*http.Request) (interface{}, error)
		expectedResponse string
	}{
		"Endpoint returning Proto": {
			endpointFunc: func(req *http.Request) (interface{}, error) {
				return endpointReturningProto(req)
			},
			expectedResponse: fixtures.GetJSONSerializedTestAlert(),
		},
		"Endpoint returning JSON Marshalable": {
			endpointFunc: func(req *http.Request) (interface{}, error) {
				return endpointReturningJSONMarshalable(req)
			},
			expectedResponse: `{"name": "jsonMarshalable"}`,
		},
	} {
		t.Run(testName, func(t *testing.T) {
			server := httptest.NewServer(RESTHandler(testCase.endpointFunc))
			defer server.Close()
			resp, err := http.Get(server.URL)
			assert.NoError(t, err)
			respBody := resp.Body
			defer func() { _ = respBody.Close() }()
			respData, err := io.ReadAll(respBody)
			assert.NoError(t, err)
			assert.JSONEq(t, testCase.expectedResponse, string(respData))
		})
	}
}
