// Package variantbar is a fixture for cache-go-dependencies-regression.yaml.
// See tools/cicachetest/README.md.
package variantbar

// Value must stay byte-for-byte length-compatible with variantfoo.Value.
func Value() int { return 2 }
