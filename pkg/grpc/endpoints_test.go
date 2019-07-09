package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPortsConfig_Validate_Valid(t *testing.T) {
	t.Parallel()

	validConfigs := []EndpointsConfig{
		{},
		{
			MultiplexedEndpoints: []string{":8080"},
			HTTPEndpoints:        []string{":8081"},
			GRPCEndpoints:        []string{":8082"},
		},
		{
			MultiplexedEndpoints: []string{":http"},
			HTTPEndpoints:        []string{"127.0.0.1:81"},
		},
	}

	for _, cfg := range validConfigs {
		assert.NoErrorf(t, cfg.Validate(), "expected config %+v to pass validation", cfg)
	}
}

func TestPortsConfig_Validate_Invalid(t *testing.T) {
	t.Parallel()

	validConfigs := []EndpointsConfig{
		{
			MultiplexedEndpoints: []string{"8080"},
		},
		{
			MultiplexedEndpoints: []string{":whatever"},
		},
		{
			MultiplexedEndpoints: []string{"localhost:whatever"},
		},
	}

	for _, cfg := range validConfigs {
		assert.Errorf(t, cfg.Validate(), "expected config %+v to fail validation", cfg)
	}
}

func TestPortsConfig_AddFromParsedSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		spec     string
		expected EndpointsConfig
	}{
		{
			spec:     "",
			expected: EndpointsConfig{},
		},
		{
			spec: "8080",
			expected: EndpointsConfig{
				MultiplexedEndpoints: []string{":8080"},
			},
		},
		{
			spec: ":8080, http@127.0.0.1:8081",
			expected: EndpointsConfig{
				MultiplexedEndpoints: []string{":8080"},
				HTTPEndpoints:        []string{"127.0.0.1:8081"},
			},
		},
		{
			spec: "http @ http, grpc @ 127.0.0.1:https, grpc@:8082, localhost:8080, http@127.0.0.1:10080",
			expected: EndpointsConfig{
				MultiplexedEndpoints: []string{"localhost:8080"},
				HTTPEndpoints:        []string{":http", "127.0.0.1:10080"},
				GRPCEndpoints:        []string{"127.0.0.1:https", ":8082"},
			},
		},
	}

	for _, testCase := range cases {
		tc := testCase
		t.Run(tc.spec, func(t *testing.T) {
			var cfg EndpointsConfig
			assert.NoError(t, cfg.AddFromParsedSpec(tc.spec), "unexpected error parsing spec %q", tc.spec)
			assert.Equal(t, tc.expected, cfg)
		})
	}
}
