package httputil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs2xxStatusCode(t *testing.T) {
	t.Parallel()

	cases := map[bool][]int{
		true: {
			http.StatusOK,
			http.StatusAccepted,
			http.StatusAlreadyReported,
			http.StatusCreated,
			http.StatusIMUsed,
		},
		false: {
			http.StatusFound,
			http.StatusMultipleChoices,
			http.StatusTemporaryRedirect,
			http.StatusUseProxy,
			http.StatusBadGateway,
			http.StatusBadRequest,
			http.StatusConflict,
			http.StatusForbidden,
			http.StatusGone,
			http.StatusNotFound,
			http.StatusProcessing,
			http.StatusNotImplemented,
		},
	}

	for expectedResult, codes := range cases {
		for _, code := range codes {
			assert.Equal(t, expectedResult, Is2xxStatusCode(code), "expected Is2xxStatusCode(%d) to be %v", code, expectedResult)
		}
	}
}

func TestIs2xxOr3xxStatusCode(t *testing.T) {
	t.Parallel()

	cases := map[bool][]int{
		true: {
			http.StatusOK,
			http.StatusAccepted,
			http.StatusAlreadyReported,
			http.StatusCreated,
			http.StatusFound,
			http.StatusIMUsed,
			http.StatusMultipleChoices,
			http.StatusTemporaryRedirect,
			http.StatusUseProxy,
		},
		false: {
			http.StatusBadGateway,
			http.StatusBadRequest,
			http.StatusConflict,
			http.StatusForbidden,
			http.StatusGone,
			http.StatusNotFound,
			http.StatusProcessing,
			http.StatusNotImplemented,
		},
	}

	for expectedResult, codes := range cases {
		for _, code := range codes {
			assert.Equal(t, expectedResult, Is2xxOr3xxStatusCode(code), "expected Is2xxOr3xxStatusCode(%d) to be %v", code, expectedResult)
		}
	}
}
