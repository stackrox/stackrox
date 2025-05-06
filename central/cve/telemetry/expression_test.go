package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_expression_match(t *testing.T) {
	testLabels := map[Label]string{
		"string": "value",
		"number": "3.4",
		"bool":   "false",
		"empty":  "",
	}

	labels := func(label Label) string {
		return testLabels[label]
	}

	t.Run("test label and value", func(t *testing.T) {
		value, ok := (&expression{"string", "=", "value"}).match(labels)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value, ok = (&expression{"number", ">=", "1.0"}).match(labels)
		assert.True(t, ok)
		assert.Equal(t, "3.4", value)
	})

	t.Run("test missing label", func(t *testing.T) {
		value, ok := (&expression{"nonexistent", "=", "value"}).match(labels)
		assert.Equal(t, "", value)
		assert.False(t, ok)
	})
	t.Run("test empty label value", func(t *testing.T) {
		value, ok := (&expression{"empty", "=", "value"}).match(labels)
		assert.Equal(t, "", value)
		assert.False(t, ok)

		value, ok = (*expression)(nil).match(labels)
		assert.Equal(t, "", value)
		assert.True(t, ok)
	})
	t.Run("test expression with only label", func(t *testing.T) {
		value, ok := (&expression{"string", "", ""}).match(labels)
		assert.Equal(t, "value", value)
		assert.True(t, ok)
	})

	t.Run("=", func(t *testing.T) {
		for _, expr := range []*expression{
			{"string", "=", "value"},
			{"string", "=", "*alu?"},
			{"number", "=", "3.40"},
			{"bool", "=", "false"},
		} {
			_, ok := expr.match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []*expression{
			{"string", "=", "value1"},
			{"string", "=", "*2"},
			{"number", "=", "3.40.1"},
			{"bool", "=", "true"},
		} {
			_, ok := expr.match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("!=", func(t *testing.T) {
		for _, expr := range []*expression{
			{"string", "!=", "value1"},
			{"string", "!=", "*2"},
			{"number", "!=", "3.5"},
			{"bool", "!=", "true"},
		} {
			_, ok := expr.match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []*expression{
			{"string", "!=", "value"},
			{"string", "!=", "*alu?"},
			{"number", "!=", "3.4"},
			{"bool", "!=", "false"},
		} {
			_, ok := expr.match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run(">", func(t *testing.T) {
		for _, expr := range []*expression{
			{"number", ">", "3.0"},
			{"number", ">", "0"},
			{"number", ">", "-1"},
			{"number", ">=", "3.4"},
		} {
			_, ok := expr.match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []*expression{
			{"number", ">", "3.4"},
			{"number", ">", "34"},
			{"number", ">", "+4334"},
			{"number", ">=", "3.41"},
		} {
			_, ok := expr.match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("<", func(t *testing.T) {
		for _, expr := range []*expression{
			{"number", "<", "3.41"},
			{"number", "<", "34"},
			{"number", "<", "+4334"},
			{"number", "<=", "3.4"},
		} {
			_, ok := expr.match(labels)
			assert.True(t, ok, expr)
		}
		for _, expr := range []*expression{
			{"number", "<", "3.4"},
			{"number", "<", "0"},
			{"number", "<", "-1"},
			{"number", "<=", "3.3"},
		} {
			_, ok := expr.match(labels)
			assert.False(t, ok, expr)
		}
	})

	t.Run("string comparison", func(t *testing.T) {
		for _, expr := range []*expression{
			{"number", "<", "3.a"},
			{"number", "!=", "3,4"},
			{"string", ">", "val"},
			{"string", "<=", "value1"},
			{"number", "<", ">3.4"},
		} {
			_, ok := expr.match(labels)
			assert.True(t, ok, expr)
		}
	})
}
