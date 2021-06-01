package translation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
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
	assert.Equal(t, chartutil.Values{"flag": true}, build(t, &v))
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

	assert.Equal(t, chartutil.Values{
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
	assert.Equal(t, chartutil.Values{"foo": "bar"}, build(t, &v))
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

	assert.Equal(t, chartutil.Values{
		"parent-foo": "foo1",
		"child": chartutil.Values{
			"child-foo": "foo6",
		},
	}, build(t, &parent))
}

func TestAddChildEmpty(t *testing.T) {
	empty, v := NewValuesBuilder(), NewValuesBuilder()
	v.SetStringValue("foo", "bar")
	v.AddChild("nil-child", nil)
	v.AddChild("empty-child", &empty)
	assert.Equal(t, chartutil.Values{"foo": "bar"}, build(t, &v))
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

	pullPolicy := v1.PullAlways
	v.SetPullPolicy("pull-policy", &pullPolicy)
	v.SetPullPolicy("nil-pull-policy", nil)

	stringMap := map[string]string{"string-key": "string-value"}
	v.SetStringMap("string-map", stringMap)
	v.SetStringMap("nil-string-map", nil)
	v.SetStringMap("empty-string-map", map[string]string{})

	resources := v1.ResourceList{v1.ResourceCPU: resource.Quantity{Format: "6"}}
	v.SetResourceList("resources", resources)
	v.SetResourceList("nil-resources", nil)
	v.SetResourceList("empty-resources", v1.ResourceList{})

	values := chartutil.Values{"chartutil-key": "chartutil-anything"}
	v.SetChartutilValues("chartutil-values", values)
	v.SetChartutilValues("nil-chartutil-values", nil)
	v.SetChartutilValues("empty-chartutil-values", chartutil.Values{})

	valuesSlice := []chartutil.Values{{"chartutil-1": 1}, {"chartutil-2": 2}}
	v.SetChartutilValuesSlice("chartutil-values-slice", valuesSlice)
	v.SetChartutilValuesSlice("nil-chartutil-values-slice", nil)
	v.SetChartutilValuesSlice("empty-chartutil-values-slice", []chartutil.Values{})

	assert.Equal(t, chartutil.Values{
		"bool-pointer":           true,
		"bool":                   true,
		"string-pointer":         "freedom",
		"empty-string-pointer":   "",
		"string":                 "freedom",
		"empty-string":           "",
		"pull-policy":            "Always",
		"string-map":             map[string]string{"string-key": "string-value"},
		"resources":              v1.ResourceList{v1.ResourceCPU: resource.Quantity{Format: "6"}},
		"chartutil-values":       chartutil.Values{"chartutil-key": "chartutil-anything"},
		"chartutil-values-slice": []chartutil.Values{{"chartutil-1": 1}, {"chartutil-2": 2}},
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
			v.SetResourceList(key, v1.ResourceList{v1.ResourcePods: resource.Quantity{Format: "14"}})
		},
		"chartutil-values": func(v *ValuesBuilder) {
			v.SetChartutilValues(key, chartutil.Values{"foo": 100500})
		},
		"chartutil-values-slice": func(v *ValuesBuilder) {
			v.SetChartutilValuesSlice(key, []chartutil.Values{{"bar": -1}})
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

func build(t *testing.T, b *ValuesBuilder) chartutil.Values {
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
