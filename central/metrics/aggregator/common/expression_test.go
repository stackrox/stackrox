package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_expression_match(t *testing.T) {
	labels := map[Label]string{
		"string": "value",
		"number": "3.4",
		"bool":   "false",
		"empty":  "",
	}

	t.Run("test label and value", func(t *testing.T) {
		value := labels["string"]
		ok := (&Expression{"=", "value"}).match(value)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value = labels["number"]
		ok = (&Expression{">=", "1.0"}).match(value)
		assert.True(t, ok)
		assert.Equal(t, "3.4", value)
	})

	t.Run("test missing label", func(t *testing.T) {
		value := labels["nonexistent"]
		ok := (&Expression{"=", "value"}).match(value)
		assert.Equal(t, "", value)
		assert.False(t, ok)
	})
	t.Run("test empty label value", func(t *testing.T) {
		value := labels["empty"]
		ok := (&Expression{"=", "value"}).match(value)
		assert.Equal(t, "", value)
		assert.False(t, ok)

		value = labels["label"]
		ok = (*Expression)(nil).match(value)
		assert.Equal(t, "", value)
		assert.True(t, ok)
	})
	t.Run("test expression with only label", func(t *testing.T) {
		value := labels["string"]
		ok := (&Expression{"", ""}).match(value)
		assert.Equal(t, "value", value)
		assert.True(t, ok)
	})

	type testCase struct {
		label Label
		expr  Expression

		match bool
	}

	for i, c := range []testCase{
		{"string", Expression{"=", "value"}, true},
		{"string", Expression{"=", "*alu?"}, true},
		{"number", Expression{"=", "3.40"}, true},
		{"bool", Expression{"=", "false"}, true},

		{"string", Expression{"=", "value1"}, false},
		{"string", Expression{"=", "*2"}, false},
		{"number", Expression{"=", "3.40.1"}, false},
		{"bool", Expression{"=", "true"}, false},

		{"string", Expression{"!=", "value1"}, true},
		{"string", Expression{"!=", "*2"}, true},
		{"number", Expression{"!=", "3.5"}, true},
		{"bool", Expression{"!=", "true"}, true},

		{"string", Expression{"!=", "value"}, false},
		{"string", Expression{"!=", "*alu?"}, false},
		{"number", Expression{"!=", "3.4"}, false},
		{"bool", Expression{"!=", "false"}, false},

		{"number", Expression{">", "3.0"}, true},
		{"number", Expression{">", "0"}, true},
		{"number", Expression{">", "-1"}, true},
		{"number", Expression{">=", "3.4"}, true},

		{"number", Expression{">", "3.4"}, false},
		{"number", Expression{">", "34"}, false},
		{"number", Expression{">", "+4334"}, false},
		{"number", Expression{">=", "3.41"}, false},

		{"number", Expression{"<", "3.41"}, true},
		{"number", Expression{"<", "34"}, true},
		{"number", Expression{"<", "+4334"}, true},
		{"number", Expression{"<=", "3.4"}, true},

		{"number", Expression{"<", "3.4"}, false},
		{"number", Expression{"<", "0"}, false},
		{"number", Expression{"<", "-1"}, false},
		{"number", Expression{"<=", "3.3"}, false},

		// string comparison:
		{"number", Expression{"<", "3.a"}, true},
		{"number", Expression{"!=", "3,4"}, true},
		{"string", Expression{">", "val"}, true},
		{"string", Expression{"<=", "value1"}, true},
		{"number", Expression{"<", ">3.4"}, true},
	} {
		value := labels[c.label]
		actual := c.expr.match(value)
		assert.Equal(t, c.match, actual, "test #%d %s %s", i, c.label, c.expr.String())
	}
}

func Test_validate(t *testing.T) {
	type testCase struct {
		expr Expression
		err  string
	}
	cases := []testCase{
		// NOK:
		{Expression{op: "op"}, "unknown operator in \"op\""},
		{Expression{op: "OR", arg: "arg"}, "unexpected argument in \"ORarg\""},
		{Expression{op: "="}, "missing argument in \"=\""},
		{Expression{op: "?", arg: "arg"}, "unknown operator in \"?arg\""},
		{Expression{op: "=", arg: "[a-"}, "cannot parse the argument in \"=[a-\""},
		// OK:
		{Expression{}, "empty operator"},
		{Expression{op: "OR"}, ""},
		{Expression{op: "=", arg: "arg"}, ""},
		{Expression{op: ">=", arg: "4.5"}, ""},
		{Expression{op: "=", arg: "def"}, ""},
	}
	for _, c := range cases {
		t.Run("expr: "+string(c.expr.op)+c.expr.arg, func(t *testing.T) {
			err := c.expr.validate()
			if err == nil {
				assert.Empty(t, c.err)
			} else {
				assert.Equal(t, c.err, err.Error())
			}
		})
	}
}
