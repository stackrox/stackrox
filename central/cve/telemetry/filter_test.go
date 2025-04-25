package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeExpression(t *testing.T) {
	cases := map[string]expression{
		"no_expr":      {"no_expr", "", ""},
		"a=b":          {"a", "=", "b"},
		"a!=b":         {"a", "!=", "b"},
		"b<=5":         {"b", "<=", "5"},
		"b<something":  {"b", "<", "something"},
		"b>something":  {"b", ">", "something"},
		"AbX_Ze>=5.43": {"AbX_Ze", ">=", "5.43"},
		"abc!def":      {"abc", "", "!def"},
		"=":            {"", "=", ""},
		"=arg":         {"", "=", "arg"},
		"label=":       {"label", "=", ""},
		"":             {"", "", ""},
	}
	for expr, result := range cases {
		assert.Equal(t, result, makeExpression(expr), expr)
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
		value, ok := filter(makeExpression("string=value"), labels)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value, ok = filter(makeExpression("number>=1.0"), labels)
		assert.True(t, ok)
		assert.Equal(t, "3.4", value)
	})

	t.Run("=", func(t *testing.T) {
		for _, expr := range []string{
			"string=value",
			"string=*alu?",
			"number=3.4",
			"bool=false",
		} {
			_, ok := filter(makeExpression(expr), labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"string=value1",
			"string=*2",
			"number=3.40", // Compared as string.
			"bool=true",
		} {
			_, ok := filter(makeExpression(expr), labels)
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
			_, ok := filter(makeExpression(expr), labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"string!=value",
			"string!=*alu?",
			"number!=3.4",
			"bool!=false",
		} {
			_, ok := filter(makeExpression(expr), labels)
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
			_, ok := filter(makeExpression(expr), labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"number>3.4",
			"number>34",
			"number>+4334",
			"number>=3.41",
		} {
			_, ok := filter(makeExpression(expr), labels)
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
			_, ok := filter(makeExpression(expr), labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"number<3.4",
			"number<0",
			"number<-1",
			"number<=3.3",
		} {
			_, ok := filter(makeExpression(expr), labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("bad op", func(t *testing.T) {
		for _, expr := range []string{
			"number<a",
			"number<",
			"<34",
			"number<>3.4",
			"number~a",
		} {
			_, ok := filter(makeExpression(expr), labels)
			assert.False(t, ok, expr)
		}
	})
}
