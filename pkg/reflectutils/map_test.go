package reflectutils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTypedMap(t *testing.T) {
	t.Parallel()

	genericMap := map[interface{}]interface{}{
		"foo": 42,
		"bar": 37,
		"baz": 1337,
	}

	typedMap := ToTypedMap(genericMap, reflect.TypeOf(""), reflect.TypeOf(0)).(map[string]int)

	assert.Equal(t, map[string]int{
		"foo": 42,
		"bar": 37,
		"baz": 1337,
	}, typedMap)
}
