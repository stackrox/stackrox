package errox

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSensitive(t *testing.T) {
	dnsError := &net.DNSError{Err: "DNS error", Name: "localhost", Server: "127.0.0.1"}

	tests := map[string]struct {
		opts               []sensitiveErrorOption
		expectPublic       string
		expectSensitive    string
		expectNil          bool
		expectNotSensitive bool
	}{
		"nil with no options": {expectNil: true, expectNotSensitive: true},
		"not nil, not sensitive with only public message": {
			opts: []sensitiveErrorOption{
				WithPublicMessage("message")},
			expectPublic:       "message",
			expectSensitive:    "message",
			expectNotSensitive: true,
		},
		"sensitive error with public message": {
			opts: []sensitiveErrorOption{
				WithPublicMessage("public"),
				WithSensitive(dnsError),
			},
			expectPublic:    "public: lookup: DNS error",
			expectSensitive: "lookup localhost on 127.0.0.1: DNS error",
		},
		"formatted sensitive": {
			opts: []sensitiveErrorOption{
				WithSensitivef("format %q", "value")},
			expectPublic:    "",
			expectSensitive: "format \"value\"",
		},
		"formatted sensitive with public error": {
			opts: []sensitiveErrorOption{
				WithPublicError("public", errors.New("message")),
				WithSensitivef("secret %v", "1.2.3.4")},
			expectPublic:    "public: message",
			expectSensitive: "secret 1.2.3.4: message",
		},
		"sensitive with public err": {
			opts: []sensitiveErrorOption{
				WithPublicError("public", errors.New("error")),
				WithSensitive(errors.New("secret"))},
			expectPublic:    "public: error",
			expectSensitive: "secret: error",
		},
		"sensitive in public": {
			opts: []sensitiveErrorOption{
				WithSensitivef("sensitive"),
				WithPublicError("oops", MakeSensitive("public", dnsError)),
			},
			expectPublic:    "oops: public",
			expectSensitive: "sensitive: lookup localhost on 127.0.0.1: DNS error",
		},
		"sensitive in public, different order, same result": {
			opts: []sensitiveErrorOption{
				WithPublicError("oops", MakeSensitive("public", dnsError)),
				WithSensitivef("sensitive"),
			},
			expectPublic:    "oops: public",
			expectSensitive: "sensitive: lookup localhost on 127.0.0.1: DNS error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := NewSensitive(test.opts...)
			if test.expectNil {
				assert.Nil(t, err)
			} else {
				require.NotNil(t, err)
				assert.Equal(t, test.expectPublic, err.Error())
			}
			assert.Equal(t, test.expectSensitive, UnconcealSensitive(err))
			var serr SensitiveError
			assert.NotEqual(t, test.expectNotSensitive, errors.As(err, &serr))
		})
	}
}
