// Package testcache is a fixture for cache-go-dependencies-regression.yaml.
// See tools/cicachetest/README.md.
package testcache

import (
	"os"
	"testing"
)

// TestReadsFixture reads testdata at runtime, which is what makes Go's test
// result cache key on (path, size, mtime) for this test instead of pure
// content hashing. Never change this test or its testdata without also
// checking cache-go-dependencies-regression.yaml's Assertion 3.
func TestReadsFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/fixture.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "cicachetest-fixture\n" {
		t.Fatalf("unexpected testdata content: %q", data)
	}
}
