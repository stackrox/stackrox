package imageintegrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		endpoint string

		url string
	}{
		{
			endpoint: "https://docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "index.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "registry-1.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://registry-1.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://index.docker.io",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://index.docker.io/v1",
			url:      "https://registry-1.docker.io",
		},
		{
			endpoint: "https://myregistry.hello.io",
			url:      "https://myregistry.hello.io",
		},
		{
			endpoint: "myregistry.hello.io",
			url:      "https://myregistry.hello.io",
		},
		{
			endpoint: "myregistry.hello.io/v1/randompage",
			url:      "https://myregistry.hello.io",
		},
		{
			endpoint: "http://myregistry.hello.io/v1/randompage",
			url:      "http://myregistry.hello.io",
		},
		{
			endpoint: "http://myregistry.hello.io:5000/v1/randompage",
			url:      "http://myregistry.hello.io:5000",
		},
	}
	for _, c := range cases {
		t.Run(c.endpoint, func(t *testing.T) {
			url := parseEndpointForURL(c.endpoint)
			assert.Equal(t, c.url, url)
		})
	}
}
