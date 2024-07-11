package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoalesceTables_LeftToRight(t *testing.T) {
	dst := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "baz",
		},
	}
	src1 := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "nope",
			"qux": "quux",
		},
	}
	src2 := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "nope nope",
			"qux": "NOPE",
		},
		"quuz": "corge",
	}

	result := CoalesceTables(dst, src1, src2)

	expected := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "baz",
			"qux": "quux",
		},
		"quuz": "corge",
	}

	assert.Equal(t, expected, result)
}
