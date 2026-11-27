package main

import (
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWriteConfig(t *testing.T) {
	const (
		registry = "quay.io/stackrox-io"
		overlay  = "5.1.x-562-g87fca7b4db"
		base     = "5.1.x-585-g972501b80b"
	)
	allOnOverlay := map[image]string{
		mainImage:           overlay,
		centralDBImage:      overlay,
		scannerV4Image:      overlay,
		scannerV4DBImage:    overlay,
		operatorBundleImage: overlay,
	}

	cases := map[string]struct {
		tags   map[image]string  // merged over allOnOverlay
		want   map[string]string // YAML path -> value
		absent []string          // YAML paths
	}{
		"all images on overlay": {
			want: map[string]string{
				"roxie.imageRegistry": registry,
				"roxie.version":       overlay,
			},
			absent: []string{
				"operator.operandTagOverrides.main",
				"operator.operandTagOverrides.scanner-v4",
				"operator.operandTagOverrides.collector",
				"operator.operandTagOverrides.fact",
			},
		},
		"image falling back to base is overridden": {
			tags: map[image]string{scannerV4Image: base},
			want: map[string]string{"operator.operandTagOverrides.scanner-v4": base},
			absent: []string{
				"operator.operandTagOverrides.main",
				"operator.operandTagOverrides.scanner-v4-db",
			},
		},
		"base operator bundle overrides overlay images": {
			tags: map[image]string{operatorBundleImage: base},
			want: map[string]string{
				"roxie.version":                              base,
				"operator.operandTagOverrides.main":          overlay,
				"operator.operandTagOverrides.scanner-v4-db": overlay,
			},
		},
		"explicit collector and fact tags": {
			tags: map[image]string{collectorImage: "3.25.0", factImage: "0.3.0"},
			want: map[string]string{
				"operator.operandTagOverrides.collector": "3.25.0",
				"operator.operandTagOverrides.fact":      "0.3.0",
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			tags := maps.Clone(allOnOverlay)
			maps.Copy(tags, c.tags)

			var out strings.Builder
			require.NoError(t, writeConfig(&out, registry, tags, NewBaseTag(registry, base, overlay), overlay))
			var doc map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(out.String()), &doc))

			for path, want := range c.want {
				got, ok := lookupYAMLPath(doc, path)
				if assert.True(t, ok, "missing %s", path) {
					assert.Equal(t, want, got, path)
				}
			}
			for _, path := range c.absent {
				_, ok := lookupYAMLPath(doc, path)
				assert.False(t, ok, "unexpected %s", path)
			}
		})
	}
}

// lookupYAMLPath resolves a dot-separated path in a decoded YAML document.
func lookupYAMLPath(doc map[string]any, path string) (any, bool) {
	var node any = doc
	for _, key := range strings.Split(path, ".") {
		m, ok := node.(map[string]any)
		if !ok {
			return nil, false
		}
		if node, ok = m[key]; !ok {
			return nil, false
		}
	}
	return node, true
}
