package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_condition_match(t *testing.T) {
	labels := map[Label]string{
		"string": "value",
		"number": "3.4",
		"bool":   "false",
		"empty":  "",
	}

	t.Run("test label and value", func(t *testing.T) {
		value := labels["string"]
		ok := (&Condition{"=", "value"}).match(value)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value = labels["number"]
		ok = (&Condition{">=", "1.0"}).match(value)
		assert.True(t, ok)
		assert.Equal(t, "3.4", value)
	})

	t.Run("test missing label", func(t *testing.T) {
		value := labels["nonexistent"]
		ok := (&Condition{"=", "value"}).match(value)
		assert.Equal(t, "", value)
		assert.False(t, ok)
	})
	t.Run("test empty label value", func(t *testing.T) {
		value := labels["empty"]
		ok := (&Condition{"=", "value"}).match(value)
		assert.Equal(t, "", value)
		assert.False(t, ok)

		value = labels["label"]
		ok = (*Condition)(nil).match(value)
		assert.Equal(t, "", value)
		assert.True(t, ok)
	})
	t.Run("test condition with only label", func(t *testing.T) {
		value := labels["string"]
		ok := (&Condition{"", ""}).match(value)
		assert.Equal(t, "value", value)
		assert.True(t, ok)
	})

	type testCase struct {
		label Label
		cond  Condition

		match bool
	}

	for i, c := range []testCase{
		{"string", Condition{"=", "value"}, true},
		{"string", Condition{"=", "*alu?"}, true},
		{"number", Condition{"=", "3.40"}, true},
		{"bool", Condition{"=", "false"}, true},

		{"string", Condition{"=", "value1"}, false},
		{"string", Condition{"=", "*2"}, false},
		{"number", Condition{"=", "3.40.1"}, false},
		{"bool", Condition{"=", "true"}, false},

		{"string", Condition{"!=", "value1"}, true},
		{"string", Condition{"!=", "*2"}, true},
		{"number", Condition{"!=", "3.5"}, true},
		{"bool", Condition{"!=", "true"}, true},

		{"string", Condition{"!=", "value"}, false},
		{"string", Condition{"!=", "*alu?"}, false},
		{"number", Condition{"!=", "3.4"}, false},
		{"bool", Condition{"!=", "false"}, false},

		{"number", Condition{">", "3.0"}, true},
		{"number", Condition{">", "0"}, true},
		{"number", Condition{">", "-1"}, true},
		{"number", Condition{">=", "3.4"}, true},

		{"number", Condition{">", "3.4"}, false},
		{"number", Condition{">", "34"}, false},
		{"number", Condition{">", "+4334"}, false},
		{"number", Condition{">=", "3.41"}, false},

		{"number", Condition{"<", "3.41"}, true},
		{"number", Condition{"<", "34"}, true},
		{"number", Condition{"<", "+4334"}, true},
		{"number", Condition{"<=", "3.4"}, true},

		{"number", Condition{"<", "3.4"}, false},
		{"number", Condition{"<", "0"}, false},
		{"number", Condition{"<", "-1"}, false},
		{"number", Condition{"<=", "3.3"}, false},

		// string comparison:
		{"number", Condition{"<", "3.a"}, true},
		{"number", Condition{"!=", "3,4"}, true},
		{"string", Condition{">", "val"}, true},
		{"string", Condition{"<=", "value1"}, true},
		{"number", Condition{"<", ">3.4"}, true},
	} {
		value := labels[c.label]
		actual := c.cond.match(value)
		assert.Equal(t, c.match, actual, "test #%d %s %s", i, c.label, c.cond.String())
	}
}

func Test_validate(t *testing.T) {
	type testCase struct {
		cond Condition
		err  string
	}
	cases := []testCase{
		// NOK:
		{Condition{op: "op"}, `operator in "op" is not one of ["=" "!=" ">" ">=" "<" "<=" "OR"]`},
		{Condition{op: "OR", arg: "arg"}, "unexpected argument in \"ORarg\""},
		{Condition{op: "="}, "missing argument in \"=\""},
		{Condition{op: "?", arg: "arg"}, `operator in "?arg" is not one of ["=" "!=" ">" ">=" "<" "<=" "OR"]`},
		{Condition{op: "=", arg: "[a-"}, "cannot parse the argument in \"=[a-\""},
		// OK:
		{Condition{}, "empty operator"},
		{Condition{op: "OR"}, ""},
		{Condition{op: "=", arg: "arg"}, ""},
		{Condition{op: ">=", arg: "4.5"}, ""},
		{Condition{op: "=", arg: "def"}, ""},
	}
	for _, c := range cases {
		t.Run("cond: "+string(c.cond.op)+c.cond.arg, func(t *testing.T) {
			err := c.cond.validate()
			if err == nil {
				assert.Empty(t, c.err)
			} else {
				assert.Equal(t, c.err, err.Error())
			}
		})
	}
}
