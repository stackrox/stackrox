//go:build test_all

package protocompat

import (
	"testing"

	"github.com/stackrox/rox/pkg/protoassert"
)

func TestEmpty(t *testing.T) {
	refEmpty := &Empty{}

	protoassert.Equal(t, refEmpty, ProtoEmpty())
}
