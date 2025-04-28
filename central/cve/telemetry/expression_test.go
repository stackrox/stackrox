package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeExpression(t *testing.T) {
	cases := map[string]*expression{
		"no_expr":      {"no_expr", "", ""},
		"a=b":          {"a", "=", "b"},
		"a!=b":         {"a", "!=", "b"},
		"b<=5":         {"b", "<=", "5"},
		"b<something":  {"b", "<", "something"},
		"b>something":  {"b", ">", "something"},
		"AbX_Ze>=5.43": {"AbX_Ze", ">=", "5.43"},
		"abc!def":      nil,
		"=":            nil,
		"=arg":         nil,
		"label=":       nil,
		"":             nil,
	}
	for expr, result := range cases {
		assert.Equal(t, result, makeExpression(expr), expr)
	}
}

func Test_expression_match(t *testing.T) {
	testLabels := map[string]string{
		"string": "value",
		"number": "3.4",
		"bool":   "false",
		"empty":  "",
	}

	labels := func(label string) string {
		return testLabels[label]
	}

	t.Run("test label and value", func(t *testing.T) {
		value, ok := makeExpression("string=value").match(labels)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value, ok = makeExpression("number>=1.0").match(labels)
		assert.True(t, ok)
		assert.Equal(t, "3.4", value)
	})

	t.Run("test missing label", func(t *testing.T) {
		value, ok := makeExpression("nonexistent=value").match(labels)
		assert.Equal(t, "", value)
		assert.False(t, ok)
	})
	t.Run("test empty label value", func(t *testing.T) {
		value, ok := makeExpression("empty=value").match(labels)
		assert.Equal(t, "", value)
		assert.False(t, ok)

		value, ok = makeExpression("empty=").match(labels)
		assert.Equal(t, "", value)
		assert.False(t, ok)
	})
	t.Run("test expression with only label", func(t *testing.T) {
		value, ok := makeExpression("string").match(labels)
		assert.Equal(t, "value", value)
		assert.True(t, ok)
	})

	t.Run("=", func(t *testing.T) {
		for _, expr := range []string{
			"string=value",
			"string=*alu?",
			"number=3.40",
			"bool=false",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"string=value1",
			"string=*2",
			"number=3.40.1",
			"bool=true",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("!=", func(t *testing.T) {
		for _, expr := range []string{
			"string!=value1",
			"string!=*2",
			"number!=3.5",
			"bool!=true",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"string!=value",
			"string!=*alu?",
			"number!=3.4",
			"bool!=false",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run(">", func(t *testing.T) {
		for _, expr := range []string{
			"number>3.0",
			"number>0",
			"number>-1",
			"number>=3.4",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"number>3.4",
			"number>34",
			"number>+4334",
			"number>=3.41",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("<", func(t *testing.T) {
		for _, expr := range []string{
			"number<3.41",
			"number<34",
			"number<+4334",
			"number<=3.4",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"number<3.4",
			"number<0",
			"number<-1",
			"number<=3.3",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("string comparison", func(t *testing.T) {
		for _, expr := range []string{
			"number<3.a",
			"number!=3,4",
			"string>val",
			"string<=value1",
			"number<>3.4", // {"number", "<", ">3.4"}
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.True(t, ok, expr)
		}
	})

	t.Run("bad op", func(t *testing.T) {
		for _, expr := range []string{
			"number<",
			"<34",
			"number~a",
		} {
			_, ok := makeExpression(expr).match(labels)
			assert.False(t, ok, expr)
		}
	})
}
