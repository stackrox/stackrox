package fieldmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nestedIgnored struct {
	D string `search:"D" protobuf:"blah"`
}

type nestedSecond struct {
	C float64 `search:"C" protobuf:"blah"`
}

type nestedFirst struct {
	B            string        `search:"B" protobuf:"blah"`
	NestedSecond *nestedSecond `protobuf:"blah"`
}

type testObj struct {
	A             int            `search:"A" protobuf:"blah"`
	Nested        []*nestedFirst `protobuf:"blah"`
	NestedIgnored *nestedIgnored `protobuf:"blah" search:"-"`
}

func fieldPathToPath(path FieldPath) []string {
	out := make([]string, 0, len(path))
	for _, f := range path {
		out = append(out, f.Name)
	}
	return out
}

func TestMap(t *testing.T) {
	fieldMap := MapSearchTagsToFieldPaths((*testObj)(nil))

	convertedFieldMap := make(map[string][]string, len(fieldMap))
	for k, v := range fieldMap {
		convertedFieldMap[k] = fieldPathToPath(v)
	}
	assert.Equal(t, map[string][]string{
		"a": {"A"},
		"b": {"Nested", "B"},
		"c": {"Nested", "NestedSecond", "C"},
	}, convertedFieldMap)
}
