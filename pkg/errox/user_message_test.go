package errox

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestWithUserMessage(t *testing.T) {

	tests := map[string]struct {
		err             error
		expectedMessage string
	}{
		"nil base": {
			WithUserMessage(nil, "message"),
			"message",
		},
		"two user messages": {
			WithUserMessage(WithUserMessage(nil, "message"), "second"),
			"second: message",
		},
		"message in between": {
			errors.Wrap(WithUserMessage(errors.New("first"), "message"), "second"),
			"second: message: first",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedMessage, test.err.Error())
		})
	}
}

func TestGetUserMessage(t *testing.T) {
	tests := map[string]struct {
		err             error
		expectedMessage string
	}{
		"nil": {
			nil,
			"",
		},
		"message": {
			WithUserMessage(nil, "message"),
			"message",
		},
		"no message": {
			errors.New("no message"),
			"",
		},
		"sensitive with message": {
			errors.WithMessage(WithUserMessage(nil, "message"), "sensitive"),
			"message",
		},
		"message with sensitive": {
			WithUserMessage(errors.WithMessage(nil, "sensitive"), "message"),
			"message",
		},
		"sensitive message sensitive message": {
			err: errors.WithMessage(
				WithUserMessage(
					errors.WithMessage(
						WithUserMessage(nil, "message1"),
						"sensitive1"),
					"message2"),
				"sensitive2"),
			expectedMessage: "message2: message1",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedMessage, GetUserMessage(test.err))
		})
	}
}
