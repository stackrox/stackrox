package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_splitExpression(t *testing.T) {
	arr := func(a, b, c string) [3]string {
		return [3]string{a, b, c}
	}

	cases := map[expression][3]string{
		"no_expr":      {"no_expr", "", ""},
		"a=b":          {"a", "=", "b"},
		"a!=b":         {"a", "!=", "b"},
		"b<=5":         {"b", "<=", "5"},
		"b<something":  {"b", "<", "something"},
		"b>something":  {"b", ">", "something"},
		"AbX_Ze>=5.43": {"AbX_Ze", ">=", "5.43"},
		"abc!def":      {"abc", "", "!def"},
	}
	for expr, result := range cases {
		assert.Equal(t, result, arr(splitExpression(expr)), expr)
	}
}

func Test_filter(t *testing.T) {
	testLabels := map[string]string{
		"string": "value",
		"number": "3.4",
		"bool":   "false",
	}

	labels := func(label string) string {
		return testLabels[label]
	}

	t.Run("test label and value", func(t *testing.T) {
		label, value, ok := filter("string=value", labels)
		assert.True(t, ok)
		assert.Equal(t, "string", label)
		assert.Equal(t, "value", value)

		label, value, ok = filter("number>=1.0", labels)
		assert.True(t, ok)
		assert.Equal(t, "number", label)
		assert.Equal(t, "3.4", value)
	})

	t.Run("=", func(t *testing.T) {
		for _, expr := range []expression{
			"string=value",
			"string=*alu?",
			"number=3.4",
			"bool=false",
		} {
			_, _, ok := filter(expr, labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []expression{
			"string=value1",
			"string=*2",
			"number=3.40", // Compared as string.
			"bool=true",
		} {
			_, _, ok := filter(expr, labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("!=", func(t *testing.T) {
		for _, expr := range []expression{
			"string!=value1",
			"string!=*2",
			"number!=3.5",
			"bool!=true",
		} {
			_, _, ok := filter(expr, labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []expression{
			"string!=value",
			"string!=*alu?",
			"number!=3.4",
			"bool!=false",
		} {
			_, _, ok := filter(expr, labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run(">", func(t *testing.T) {
		for _, expr := range []expression{
			"number>3.0",
			"number>0",
			"number>-1",
			"number>=3.4",
		} {
			_, _, ok := filter(expr, labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []expression{
			"number>3.4",
			"number>34",
			"number>+4334",
			"number>=3.41",
		} {
			_, _, ok := filter(expr, labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("<", func(t *testing.T) {
		for _, expr := range []expression{
			"number<3.41",
			"number<34",
			"number<+4334",
			"number<=3.4",
		} {
			_, _, ok := filter(expr, labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []expression{
			"number<3.4",
			"number<0",
			"number<-1",
			"number<=3.3",
		} {
			_, _, ok := filter(expr, labels)
			assert.False(t, ok, expr)
		}
	})
}
