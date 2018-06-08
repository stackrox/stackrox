package kubernetes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCreateSecretTemplate(t *testing.T) {
	res := GetCreateSecretTemplate("{{.Namespace}}", "{{.Registry}}", "{{.ImagePullSecret}}")

	// naive expected
	expected := strings.Replace(secretTemplate, "{{.NamespaceVar}}", "{{.Namespace}}", -1)
	expected = strings.Replace(expected, "{{.RegistryVar}}", "{{.Registry}}", -1)
	expected = strings.Replace(expected, "{{.ImagePullSecretVar}}", "{{.ImagePullSecret}}", -1)
	assert.Equal(t, expected, res)
}

func TestResolvedRegistry(t *testing.T) {

	cases := []struct {
		name     string
		image    string
		expected string
	}{
		{image: "library/nginx", expected: "https://docker.io"},
		{image: "docker.io/library/nginx:latest", expected: "https://docker.io"},
		{image: "stackrox.io/prevent:1.10", expected: "https://stackrox.io"},
		{image: "gcr.io/project-name/prevent:1.2.3", expected: "https://gcr.io"},
		{image: "dtr.example.com/stackrox/prevent:1.2.3", expected: "https://dtr.example.com"},
		{image: "docker-default.registry.svc:5000/stackrox/prevent:1.2", expected: "https://docker-default.registry.svc:5000"},
		{image: "stackrox/prevent", expected: "https://docker.io"},
		{image: "stackrox/prevent@sha256:e5f272a79b5d7ae2c5eff121370371b623d7685fd078bd257f3ac3026457fe41", expected: "https://docker.io"},
	}

	for _, c := range cases {
		t.Run(c.image, func(t *testing.T) {
			url, err := GetResolvedRegistry(c.image)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, url)
		})
	}
}
