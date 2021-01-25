package helmtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruthiness(t *testing.T) {
	truthyValues := []interface{}{
		"foo",
		1,
		map[string]interface{}{"foo": ""},
		[]string{"bar"},
		1.0,
	}

	for _, v := range truthyValues {
		assert.Truef(t, truthiness(v), "expected value %v to be truthy", v)
	}

	falsyValues := []interface{}{
		"",
		0,
		map[string]interface{}(nil),
		map[string]interface{}{},
		[]string(nil),
		[]string{},
		0.0,
	}

	for _, v := range falsyValues {
		assert.Falsef(t, truthiness(v), "expected value %v to be falsy")
	}
}
