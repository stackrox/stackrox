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

func Test_validate(t *testing.T) {
	type testCase struct {
		expr expression
		err  string
	}
	cases := []testCase{
		// NOK:
		{expression{label: ""}, "empty label in \"\""},
		{expression{label: "a b"}, "unknown label in \"a b\""},
		{expression{label: "ab", op: "op"}, "unknown label in \"abop\""},
		{expression{label: "CVE", op: "op"}, "unknown operator in \"CVEop\""},
		{expression{label: "CVE", op: "="}, "missing argument in \"CVE=\""},
		{expression{label: "", op: "=", arg: "arg"}, "empty label in \"=arg\""},
		{expression{label: "CVE", op: "?", arg: "arg"}, "unknown operator in \"CVE?arg\""},
		{expression{label: "CVE", op: "=", arg: "[a-"}, "cannot parse the argument in \"CVE=[a-\""},
		// OK:
		{expression{label: "CVE", op: "=", arg: "arg"}, ""},
		{expression{label: "CVE", op: ">=", arg: "4.5"}, ""},
		{expression{label: "CVE", op: "=", arg: "def"}, ""},
	}
	for _, c := range cases {
		t.Run("expr: "+string(c.expr.label)+string(c.expr.op)+c.expr.arg, func(t *testing.T) {
			err := c.expr.validate()
			if err == nil {
				assert.Empty(t, c.err)
			} else {
				assert.Equal(t, c.err, err.Error())
			}
		})
	}
}
