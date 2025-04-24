package telemetry

import (
	"testing"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
)

func Test_splitKey(t *testing.T) {
	arr := func(a string, b string, c string) []string {
		return []string{a, b, c}
	}

	assert.Equal(t, []string{"no_expr", "", ""}, arr(splitExpression("no_expr")))
	assert.Equal(t, []string{"a", "=", "b"}, arr(splitExpression("a=b")))
	assert.Equal(t, []string{"a", "!=", "b"}, arr(splitExpression("a!=b")))
	assert.Equal(t, []string{"b", "<=", "5"}, arr(splitExpression("b<=5")))
	assert.Equal(t, []string{"b", "<", "something"}, arr(splitExpression("b<something")))
	assert.Equal(t, []string{"b", ">", "something"}, arr(splitExpression("b>something")))
	assert.Equal(t, []string{"AbX_Ze", ">=", "5.43"}, arr(splitExpression("AbX_Ze>=5.43")))
}

func Test_filter(t *testing.T) {
	globCache = make(map[string]glob.Glob)
	metric := map[string]string{
		"string": "value",
		"number": "3.4",
		"bool":   "false",
	}

	t.Run("=", func(t *testing.T) {
		for _, expr := range []string{
			"string=value",
			"string=*alu?",
			"number=3.4",
			"bool=false",
		} {
			_, ok := filter(expr, metric)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"string=value1",
			"string=*2",
			"number=3.40", // Compared as string.
			"bool=true",
		} {
			_, ok := filter(expr, metric)
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
			_, ok := filter(expr, metric)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"string!=value",
			"string!=*alu?",
			"number!=3.4",
			"bool!=false",
		} {
			_, ok := filter(expr, metric)
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
			_, ok := filter(expr, metric)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"number>3.4",
			"number>34",
			"number>+4334",
			"number>=3.41",
		} {
			_, ok := filter(expr, metric)
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
			_, ok := filter(expr, metric)
			assert.True(t, ok, expr)
		}
		for _, expr := range []string{
			"number<3.4",
			"number<0",
			"number<-1",
			"number<=3.3",
		} {
			_, ok := filter(expr, metric)
			assert.False(t, ok, expr)
		}
	})
}
