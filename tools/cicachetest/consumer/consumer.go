// Package consumer is a fixture for cache-go-dependencies-regression.yaml.
// See tools/cicachetest/README.md. The workflow swaps the import below
// between variantfoo and variantbar (equal-length names) without changing
// this file's byte size, mtime, or filename — reproducing the exact Go
// module-index dirHash collision that caused the golang-jwt v4->v5 incident.
package consumer

import "github.com/stackrox/stackrox/tools/cicachetest/variantfoo"

func Value() int { return variantfoo.Value() }
