package reflectutils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTypedSlice(t *testing.T) {
	t.Parallel()

	genericSlice := []interface{}{"foo", "bar", "baz"}
	strSlice := ToTypedSlice(genericSlice, reflect.TypeOf("")).([]string)
	assert.Equal(t, []string{"foo", "bar", "baz"}, strSlice)
}
