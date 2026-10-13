package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestRender(t *testing.T) {
	cases := map[string]struct{ cpu, expected string }{
		"cores": {"2", "500m"}, "decimal": {"0.75", "500m"}, "small": {"10m", "10m"}, "cap": {"500m", "500m"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			input := `kind: Deployment
metadata:
  annotations: {ports: "8080,8443", enabled: "true"}
spec:
  template:
    spec:
      containers:
      - name: app
        image: unchanged
        resources:
          requests: {cpu: "CPU", memory: 4Gi}
          limits: {cpu: "4", memory: 8Gi, ephemeral-storage: 1Gi}
      initContainers:
      - name: init
        resources:
          requests: {cpu: "4", memory: 8Gi}
          limits: {cpu: "8", memory: 16Gi}
---
kind: ConfigMap
data: {unchanged: value}
`
			var out bytes.Buffer
			require.NoError(t, render(strings.NewReader(strings.ReplaceAll(input, "CPU", tc.cpu)), &out))
			require.Contains(t, out.String(), "cpu: "+tc.expected)
			require.NotContains(t, out.String(), "cpu: \"4\"")
			require.Contains(t, out.String(), "ephemeral-storage: 1Gi")
			require.Contains(t, out.String(), "image: unchanged")
			require.Contains(t, out.String(), `"8080,8443"`)
			require.Contains(t, out.String(), `"true"`)
			require.Contains(t, out.String(), "kind: ConfigMap")
			decoder := yaml.NewDecoder(&out)
			var doc map[string]interface{}
			require.NoError(t, decoder.Decode(&doc))
			spec := object(object(object(doc["spec"])["template"])["spec"])
			for field, memory := range map[string][2]string{"containers": {"4Gi", "8Gi"}, "initContainers": {"8Gi", "16Gi"}} {
				for _, value := range spec[field].([]interface{}) {
					resources := object(object(value)["resources"])
					requests, limits := object(resources["requests"]), object(resources["limits"])
					require.Equal(t, memory[0], requests["memory"])
					require.Equal(t, memory[1], limits["memory"])
					require.NotContains(t, limits, "cpu")
				}
			}
		})
	}
}

func TestInvalidResources(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"missing":     {},
		"invalid CPU": {"resources": map[string]interface{}{"requests": map[string]interface{}{"cpu": "bad", "memory": "1Gi"}}},
	}
	for name, container := range cases {
		t.Run(name, func(t *testing.T) { require.Error(t, normalize(container)) })
	}
}

func TestNormalizePreservesOtherResources(t *testing.T) {
	cases := map[string]struct {
		input, expected string
	}{
		"no memory or limits": {
			`name: app
resources:
  requests: {cpu: "2"}
`,
			`name: app
resources:
  requests: {cpu: 500m}
`,
		},
		"memory and storage unchanged": {
			`name: app
resources:
  requests: {cpu: "50m", memory: 10Mi, ephemeral-storage: 1Gi}
  limits: {cpu: "1", memory: 2Gi, ephemeral-storage: 4Gi}
`,
			`name: app
resources:
  requests: {cpu: "50m", memory: 10Mi, ephemeral-storage: 1Gi}
  limits: {memory: 2Gi, ephemeral-storage: 4Gi}
`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var input, expected map[string]interface{}
			require.NoError(t, yaml.Unmarshal([]byte(tc.input), &input))
			require.NoError(t, yaml.Unmarshal([]byte(tc.expected), &expected))
			require.NoError(t, normalize(input))
			require.Equal(t, expected, input)
		})
	}
}
