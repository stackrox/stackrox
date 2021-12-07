package charts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaValuesStringKeyCompatibility(t *testing.T) {
	m := MetaValues{"foo": "bar"}
	assert.Equal(t, "bar", m["foo"])

	m["foo"] = "No" + "vem" + "ber"
	assert.Equal(t, "November", m["foo"])

	str := "blah"
	// Can't use string key directly, need to cast it, otherwise the code does not compile.
	m[MetaValuesKey(str)] = 6

	assert.Equal(t, 6, m["blah"])             // String constants are casted automatically.
	assert.Equal(t, 6, m[MetaValuesKey(str)]) // String variables need to be casted explicitly.
}

func TestMetaValuesToRaw(t *testing.T) {
	m := MetaValues{"foo": "bar", "baz": 6}

	raw := m.ToRaw()

	assert.Equal(t, map[string]interface{}{"baz": 6, "foo": "bar"}, raw)

	assert.NotSame(t, raw, m)

	// This might seem a bit surprising, but assert is using reflect.DeepEqual which checks if types are matching.
	assert.NotEqual(t, raw, m)
}
