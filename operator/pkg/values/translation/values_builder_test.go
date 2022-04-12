package translation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildEmpty(t *testing.T) {
	v := NewValuesBuilder()
	assert.Empty(t, build(t, &v))
}

func TestBuildNil(t *testing.T) {
	assert.Empty(t, build(t, nil))
}

func TestBuildWithValue(t *testing.T) {
	v := NewValuesBuilder()
	v.SetBoolValue("flag", true)
	assert.Equal(t, map[string]interface{}{"flag": true}, build(t, &v))
}

func TestBuildWithError(t *testing.T) {
	v := NewValuesBuilder()
	v.SetError(errors.New("mock error"))
	assertBuildError(t, &v, "mock error")
}

func TestBuildWithValueAndError(t *testing.T) {
	v := NewValuesBuilder()
	v.SetStringValue("foo", "bar")
	v.SetError(errors.New("mock error"))
	assertBuildError(t, &v, "mock error")
}

func TestSetMultipleErrors(t *testing.T) {
	v := NewValuesBuilder()
	v.SetError(errors.New("first error"))
	v.SetError(errors.New("second error"))
	v.SetError(errors.New("Nth error"))
	assertBuildError(t, &v, "first", "second", "Nth")
}

func TestAddAllFrom(t *testing.T) {
	donor, host := NewValuesBuilder(), NewValuesBuilder()
	donor.SetBoolValue("donor-flag", false)
	donor.SetStringValue("donor-string", "foo")
	host.SetStringValue("host-string", "bar")

	host.AddAllFrom(&donor)

	assert.Equal(t, map[string]interface{}{
		"host-string":  "bar",
		"donor-string": "foo",
		"donor-flag":   false,
	}, build(t, &host))
}

func TestAllFromEmpty(t *testing.T) {
	empty, v := NewValuesBuilder(), NewValuesBuilder()
	v.SetStringValue("foo", "bar")
	v.AddAllFrom(nil)
	v.AddAllFrom(&empty)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, build(t, &v))
}

func TestAddAllFromError(t *testing.T) {
	donor1, host1 := NewValuesBuilder(), NewValuesBuilder()
	donor1.SetError(errors.New("mock error 1"))
	host1.SetStringValue("host-string", "foo")
	host1.AddAllFrom(&donor1)
	assertBuildError(t, &host1, "mock error 1")

	donor2, host2 := NewValuesBuilder(), NewValuesBuilder()
	host2.SetError(errors.New("mock error 2"))
	donor2.SetStringValue("donor-string", "bar")
	host2.AddAllFrom(&donor2)
	assertBuildError(t, &host2, "mock error 2")

	donor3, host3 := NewValuesBuilder(), NewValuesBuilder()
	donor3.SetError(errors.New("mock error 3"))
	host3.SetError(errors.New("mock error 4"))
	host3.AddAllFrom(&donor3)
	assertBuildError(t, &host3, "mock error 3", "mock error 4")
}

func TestAddAllFromKeyClash(t *testing.T) {
	donor, host := NewValuesBuilder(), NewValuesBuilder()
	donor.SetBoolValue("flag1", true)
	donor.SetStringValue("clashing-key", "abc")
	host.SetBoolValue("flag2", false)
	host.SetStringValue("clashing-key", "xyz")

	host.AddAllFrom(&donor)

	assertBuildError(t, &host, "overwrite existing key \"clashing-key\"")
}

func TestAddChild(t *testing.T) {
	child, parent := NewValuesBuilder(), NewValuesBuilder()
	child.SetStringValue("child-foo", "foo6")
	parent.SetStringValue("parent-foo", "foo1")

	parent.AddChild("child", &child)

	assert.Equal(t, map[string]interface{}{
		"parent-foo": "foo1",
		"child": map[string]interface{}{
			"child-foo": "foo6",
		},
	}, build(t, &parent))
}

func TestAddChildEmpty(t *testing.T) {
	empty, v := NewValuesBuilder(), NewValuesBuilder()
	v.SetStringValue("foo", "bar")
	v.AddChild("nil-child", nil)
	v.AddChild("empty-child", &empty)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, build(t, &v))
}

func TestAddChildWithError(t *testing.T) {
	child1, parent1 := NewValuesBuilder(), NewValuesBuilder()
	child1.SetError(errors.New("mock error 1"))
	parent1.SetStringValue("parent-foo", "foo1")
	parent1.AddChild("child", &child1)
	assertBuildError(t, &parent1, "mock error 1")

	child2, parent2 := NewValuesBuilder(), NewValuesBuilder()
	child2.SetBoolValue("child-flag", true)
	parent2.SetError(errors.New("mock error 2"))
	parent2.AddChild("child", &child2)
	assertBuildError(t, &parent2, "mock error 2")

	child3, parent3 := NewValuesBuilder(), NewValuesBuilder()
	child3.SetError(errors.New("mock error 3"))
	parent3.SetError(errors.New("mock error 4"))
	parent3.AddChild("child", &child3)
	assertBuildError(t, &parent3, "mock error 3", "mock error 4")
}

func TestAddChildKeyClash(t *testing.T) {
	parent, child := NewValuesBuilder(), NewValuesBuilder()
	parent.SetStringValue("clashing-key", "not-a-child")
	child.SetStringValue("foo", "bar")

	parent.AddChild("clashing-key", &child)

	assertBuildError(t, &parent, "overwrite existing key \"clashing-key\"")
}

func TestSetValues(t *testing.T) {
	v := NewValuesBuilder()

	truth := true
	v.SetBool("bool-pointer", &truth)
	v.SetBool("nil-bool-pointer", nil)

	v.SetBoolValue("bool", truth)

	word := "freedom"
	noWord := ""
	v.SetString("string-pointer", &word)
	v.SetString("empty-string-pointer", &noWord)
	v.SetString("nil-string-pointer", nil)

	v.SetStringValue("string", word)
	v.SetStringValue("empty-string", noWord)

	numberInt32 := int32(42)
	zeroInt32 := int32(0)
	v.SetInt32("int32", &numberInt32)
	v.SetInt32("zero-int32", &zeroInt32)

	pullPolicy := v1.PullAlways
	v.SetPullPolicy("pull-policy", &pullPolicy)
	v.SetPullPolicy("nil-pull-policy", nil)

	stringSlice := []string{"string1", ""}
	v.SetStringSlice("string-slice", stringSlice)
	v.SetStringSlice("nil-string-slice", nil)
	v.SetStringSlice("empty-string-slice", []string{})

	stringMap := map[string]string{"string-key": "string-value"}
	v.SetStringMap("string-map", stringMap)
	v.SetStringMap("nil-string-map", nil)
	v.SetStringMap("empty-string-map", map[string]string{})

	resources := v1.ResourceList{v1.ResourceCPU: resource.MustParse("6")}
	v.SetResourceList("resources", resources)
	v.SetResourceList("nil-resources", nil)
	v.SetResourceList("empty-resources", v1.ResourceList{})

	values := map[string]interface{}{"chartutil-key": "chartutil-anything"}
	v.SetMap("map", values)
	v.SetMap("nil-map", nil)
	v.SetMap("empty-map", map[string]interface{}{})

	valuesSlice := []map[string]interface{}{{"chartutil-1": 1}, {"chartutil-2": 2}}
	v.SetMapSlice("map-slice", valuesSlice)
	v.SetMapSlice("nil-map-slice", nil)
	v.SetMapSlice("empty-map-slice", []map[string]interface{}{})

	assert.Equal(t, map[string]interface{}{
		"bool-pointer":         true,
		"bool":                 true,
		"int32":                float64(42),
		"zero-int32":           float64(0),
		"string-pointer":       "freedom",
		"empty-string-pointer": "",
		"string":               "freedom",
		"empty-string":         "",
		"pull-policy":          "Always",
		"string-slice":         []interface{}{"string1", ""},
		"string-map":           map[string]interface{}{"string-key": "string-value"},
		"resources":            map[string]interface{}{"cpu": "6"},
		"map":                  map[string]interface{}{"chartutil-key": "chartutil-anything"},
		"map-slice":            []interface{}{map[string]interface{}{"chartutil-1": float64(1)}, map[string]interface{}{"chartutil-2": float64(2)}},
	}, build(t, &v))
}

func TestSetClashingKey(t *testing.T) {
	const key = "clashing-key-6"

	setters := map[string]func(builder *ValuesBuilder){
		"bool-pointer": func(v *ValuesBuilder) {
			truth := true
			v.SetBool(key, &truth)
		},
		"bool": func(v *ValuesBuilder) {
			v.SetBoolValue(key, false)
		},
		"string-pointer": func(v *ValuesBuilder) {
			word := "freedom"
			v.SetString(key, &word)
		},
		"string": func(v *ValuesBuilder) {
			v.SetStringValue(key, "blah!")
		},
		"pull-policy": func(v *ValuesBuilder) {
			policy := v1.PullIfNotPresent
			v.SetPullPolicy(key, &policy)
		},
		"string-map": func(v *ValuesBuilder) {
			v.SetStringMap(key, map[string]string{"foo": "bar"})
		},
		"resources": func(v *ValuesBuilder) {
			v.SetResourceList(key, v1.ResourceList{v1.ResourcePods: resource.MustParse("14")})
		},
		"chartutil-values": func(v *ValuesBuilder) {
			v.SetMap(key, map[string]interface{}{"foo": 100500})
		},
		"chartutil-values-slice": func(v *ValuesBuilder) {
			v.SetMapSlice(key, []map[string]interface{}{{"bar": -1}})
		},
	}

	for name, setter := range setters {
		t.Run(name, func(t *testing.T) {
			v := NewValuesBuilder()
			v.SetBoolValue(key, false)
			setter(&v)
			assertBuildError(t, &v, "overwrite existing key \"clashing-key-6\"")
		})
	}
}

func TestEmptyKey(t *testing.T) {
	v := NewValuesBuilder()
	v.SetStringValue("", "whatever value")
	assertBuildError(t, &v, "attempt to set empty key")
}

func TestSetData(t *testing.T) {
	v := NewValuesBuilder()
	v.SetPathValue("root.child.another child", "test value")
	require.Empty(t, v.errors)

	assert.Equal(t, map[string]interface{}{
		"root": map[string]interface{}{
			"child": map[string]interface{}{
				"another child": "test value",
			},
		},
	}, v.data)
}

func TestSetDataDontOverwrite(t *testing.T) {
	v := NewValuesBuilder()
	v.SetPathValue("root.child", "already existent")
	require.NoError(t, v.errors.Unwrap())
	v.SetPathValue("root.child.grandchild", "fails to be written")
	require.Error(t, v.errors)
	assert.Equal(t, map[string]interface{}{
		"root": map[string]interface{}{
			"child": "already existent",
		},
	}, v.data)
}

func build(t *testing.T, b *ValuesBuilder) map[string]interface{} {
	val, err := b.Build()
	require.NoError(t, err)
	assert.NotNil(t, val)
	return val
}

func assertBuildError(t *testing.T, b *ValuesBuilder, messageParts ...string) {
	val, err := b.Build()
	assert.Nil(t, val)
	require.Error(t, err)
	for _, m := range messageParts {
		assert.Contains(t, err.Error(), m)
	}
}
