package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoalesceTables_LeftToRight(t *testing.T) {
	dst := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
		},
	}
	src1 := map[string]any{
		"foo": map[string]any{
			"bar": "nope",
			"qux": "quux",
		},
	}
	src2 := map[string]any{
		"foo": map[string]any{
			"bar": "nope nope",
			"qux": "NOPE",
		},
		"quuz": "corge",
	}

	result := CoalesceTables(dst, src1, src2)

	expected := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
			"qux": "quux",
		},
		"quuz": "corge",
	}

	assert.Equal(t, expected, result)
}
