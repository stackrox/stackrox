// Package variantfoo is a fixture for cache-go-dependencies-regression.yaml.
// See tools/cicachetest/README.md.
package variantfoo

// Value must stay byte-for-byte length-compatible with variantbar.Value.
func Value() int { return 1 }
