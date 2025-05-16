package aggregator

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
		ok := (&expression{"=", "value"}).match(value)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value = labels["number"]
		ok = (&expression{">=", "1.0"}).match(value)
		assert.True(t, ok)
		assert.Equal(t, "3.4", value)
	})

	t.Run("test missing label", func(t *testing.T) {
		value := labels["nonexistent"]
		ok := (&expression{"=", "value"}).match(value)
		assert.Equal(t, "", value)
		assert.False(t, ok)
	})
	t.Run("test empty label value", func(t *testing.T) {
		value := labels["empty"]
		ok := (&expression{"=", "value"}).match(value)
		assert.Equal(t, "", value)
		assert.False(t, ok)

		value = labels["label"]
		ok = (*expression)(nil).match(value)
		assert.Equal(t, "", value)
		assert.True(t, ok)
	})
	t.Run("test expression with only label", func(t *testing.T) {
		value := labels["string"]
		ok := (&expression{"", ""}).match(value)
		assert.Equal(t, "value", value)
		assert.True(t, ok)
	})

	type testCase struct {
		label Label
		expr  expression

		match bool
	}

	for i, c := range []testCase{
		{"string", expression{"=", "value"}, true},
		{"string", expression{"=", "*alu?"}, true},
		{"number", expression{"=", "3.40"}, true},
		{"bool", expression{"=", "false"}, true},

		{"string", expression{"=", "value1"}, false},
		{"string", expression{"=", "*2"}, false},
		{"number", expression{"=", "3.40.1"}, false},
		{"bool", expression{"=", "true"}, false},

		{"string", expression{"!=", "value1"}, true},
		{"string", expression{"!=", "*2"}, true},
		{"number", expression{"!=", "3.5"}, true},
		{"bool", expression{"!=", "true"}, true},

		{"string", expression{"!=", "value"}, false},
		{"string", expression{"!=", "*alu?"}, false},
		{"number", expression{"!=", "3.4"}, false},
		{"bool", expression{"!=", "false"}, false},

		{"number", expression{">", "3.0"}, true},
		{"number", expression{">", "0"}, true},
		{"number", expression{">", "-1"}, true},
		{"number", expression{">=", "3.4"}, true},

		{"number", expression{">", "3.4"}, false},
		{"number", expression{">", "34"}, false},
		{"number", expression{">", "+4334"}, false},
		{"number", expression{">=", "3.41"}, false},

		{"number", expression{"<", "3.41"}, true},
		{"number", expression{"<", "34"}, true},
		{"number", expression{"<", "+4334"}, true},
		{"number", expression{"<=", "3.4"}, true},

		{"number", expression{"<", "3.4"}, false},
		{"number", expression{"<", "0"}, false},
		{"number", expression{"<", "-1"}, false},
		{"number", expression{"<=", "3.3"}, false},

		// string comparison:
		{"number", expression{"<", "3.a"}, true},
		{"number", expression{"!=", "3,4"}, true},
		{"string", expression{">", "val"}, true},
		{"string", expression{"<=", "value1"}, true},
		{"number", expression{"<", ">3.4"}, true},
	} {
		value := labels[c.label]
		actual := c.expr.match(value)
		assert.Equal(t, c.match, actual, "test #%d %s %s", i, c.label, c.expr.String())
	}
}

func Test_validate(t *testing.T) {
	type testCase struct {
		expr expression
		err  string
	}
	cases := []testCase{
		// NOK:
		{expression{op: "op"}, "unknown operator in \"op\""},
		{expression{op: "OR", arg: "arg"}, "unexpected argument in \"ORarg\""},
		{expression{op: "="}, "missing argument in \"=\""},
		{expression{op: "?", arg: "arg"}, "unknown operator in \"?arg\""},
		{expression{op: "=", arg: "[a-"}, "cannot parse the argument in \"=[a-\""},
		// OK:
		{expression{}, "empty operator"},
		{expression{op: "OR"}, ""},
		{expression{op: "=", arg: "arg"}, ""},
		{expression{op: ">=", arg: "4.5"}, ""},
		{expression{op: "=", arg: "def"}, ""},
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
