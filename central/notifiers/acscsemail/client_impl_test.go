package acscsemail

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage(t *testing.T) {

	fakeTokenFunc := func() (string, error) {
		return "test-token", nil
	}

	tokenErr := errors.New("token error")
	fakeTokenErrFunc := func() (string, error) {
		return "", tokenErr
	}

	defaultMsg := AcscsMessage{
		To:         []string{"test@test.acscs-email.test"},
		RawMessage: []byte("test message content"),
	}

	defaultContext := context.Background()

	tests := map[string]struct {
		tokenFunc       func() (string, error)
		inputMessage    AcscsMessage
		expectedError   error
		ctx             context.Context
		response        *http.Response
		expectedHeader  http.Header
		expectedBodyStr string
	}{
		"error on loadToken": {
			tokenFunc:     fakeTokenErrFunc,
			expectedError: tokenErr,
			inputMessage:  defaultMsg,
			ctx:           defaultContext,
		},
		"error on invalid context": {
			tokenFunc:     fakeTokenFunc,
			expectedError: errors.New("failed to build HTTP"),
			inputMessage:  defaultMsg,
			// ctx nil causes an error on http.NewRequest
			ctx: nil,
		},
		"error on bad status code": {
			tokenFunc:     fakeTokenFunc,
			inputMessage:  defaultMsg,
			expectedError: errors.New("failed with HTTP status: 400"),
			ctx:           defaultContext,
			response: &http.Response{
				StatusCode: 400,
			},
		},
		"successful request": {
			tokenFunc:    fakeTokenFunc,
			inputMessage: defaultMsg,
			ctx:          defaultContext,
			response: &http.Response{
				StatusCode: 200,
			},
			expectedHeader: map[string][]string{
				"Content-Type":  {"application/json; charset=UTF-8"},
				"Authorization": {"Bearer test-token"},
			},
			// RawMessage is the b64 encoded value of "test message content" defined in the inputMessage
			expectedBodyStr: `{"to":["test@test.acscs-email.test"],"rawMessage":"dGVzdCBtZXNzYWdlIGNvbnRlbnQ="}`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			httpClient, actualRequest := testClient(tc.response, tc.expectedError)
			client := clientImpl{
				loadToken:  tc.tokenFunc,
				url:        "http://localhost:8080",
				httpClient: httpClient,
			}

			err := client.SendMessage(tc.ctx, tc.inputMessage)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedHeader, actualRequest.Header)
			actualBody, err := io.ReadAll(actualRequest.Body)
			require.NoError(t, err, "error parsing request body")
			assert.Equal(t, tc.expectedBodyStr, string(actualBody))
		})
	}
}

func testClient(res *http.Response, returnErr error) (*http.Client, *http.Request) {
	var receivedRequest http.Request

	client := &http.Client{
		Transport: testRoundTripper(func(req *http.Request) (*http.Response, error) {
			receivedRequest = *req
			return res, returnErr
		}),
	}

	return client, &receivedRequest
}

type testRoundTripper func(req *http.Request) (*http.Response, error)

func (t testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t(req)
}
