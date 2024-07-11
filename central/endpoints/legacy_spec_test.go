package endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLegacySpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		spec     string
		expected []EndpointConfig
	}{
		{
			spec:     "",
			expected: nil,
		},
		{
			spec: "8080",
			expected: []EndpointConfig{
				{Listen: "8080"},
			},
		},
		{
			spec: ":8080, http@127.0.0.1:8081",
			expected: []EndpointConfig{
				{Listen: ":8080"},
				{Listen: "127.0.0.1:8081", Protocols: []string{"http"}},
			},
		},
		{
			spec: "http @ http, grpc @ 127.0.0.1:https, grpc@:8082, localhost:8080, http@127.0.0.1:10080",
			expected: []EndpointConfig{
				{Listen: "http", Protocols: []string{"http"}},
				{Listen: "127.0.0.1:https", Protocols: []string{"grpc"}},
				{Listen: ":8082", Protocols: []string{"grpc"}},
				{Listen: "localhost:8080"},
				{Listen: "127.0.0.1:10080", Protocols: []string{"http"}},
			},
		},
	}

	for _, testCase := range cases {
		tc := testCase
		t.Run(tc.spec, func(t *testing.T) {
			endpointCfgs := ParseLegacySpec(tc.spec, nil)
			assert.ElementsMatch(t, tc.expected, endpointCfgs)
		})
	}
}
