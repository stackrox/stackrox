package reflectutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type SomeStruct struct {
	I int
	S SubStruct
}

type SubStruct struct {
	S string
	P *SubStruct
}

func TestDeepMergeStructs(t *testing.T) {
	for name, testCase := range map[string]struct {
		a        interface{}
		b        interface{}
		expected interface{}
	}{
		"both empty": {
			a:        SomeStruct{},
			b:        SomeStruct{},
			expected: SomeStruct{},
		},
		"a empty": {
			a:        SomeStruct{},
			b:        SomeStruct{I: 1, S: SubStruct{S: "test"}},
			expected: SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"b empty": {
			a:        SomeStruct{I: 1, S: SubStruct{S: "test"}},
			b:        SomeStruct{},
			expected: SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"both non-empty": {
			a:        SomeStruct{I: 1},
			b:        SomeStruct{S: SubStruct{S: "test"}},
			expected: SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"preserves indirection": {
			a:        &SomeStruct{I: 1},
			b:        &SomeStruct{S: SubStruct{S: "test"}},
			expected: &SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"nested overwrite with b": {
			a:        SomeStruct{S: SubStruct{S: "from a"}},
			b:        SomeStruct{S: SubStruct{S: "from b"}},
			expected: SomeStruct{S: SubStruct{S: "from b"}},
		},
		"nil pointer only in a": {
			a:        SomeStruct{S: SubStruct{S: "from a", P: &SubStruct{S: "inner"}}},
			b:        SomeStruct{S: SubStruct{S: "from b"}},
			expected: SomeStruct{S: SubStruct{S: "from b", P: &SubStruct{S: "inner"}}},
		},
		"nil pointer only in b": {
			a:        SomeStruct{S: SubStruct{S: "from a"}},
			b:        SomeStruct{S: SubStruct{S: "from b", P: &SubStruct{S: "inner"}}},
			expected: SomeStruct{S: SubStruct{S: "from b", P: &SubStruct{S: "inner"}}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			merged := DeepMergeStructs(testCase.a, testCase.b)
			assert.Equal(t, testCase.expected, merged)
		})
	}
}
