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
	} {
		value := labels[c.label]
		actual := c.cond.match(value)
		assert.Equal(t, c.match, actual, "test #%d %s %s", i, c.label, c.cond.String())
	}
}

func TestMustMakeCondition(t *testing.T) {
	assert.Panics(t, func() { _ = MustMakeCondition("x", "y") })
}

func Test_validate(t *testing.T) {
	type testCase struct {
		cond Condition
		err  string
	}
	cases := []testCase{
		// NOK:
		{Condition{op: "op"}, `operator in "op" is not one of ["="]`},
		{Condition{op: "="}, "missing argument in \"=\""},
		{Condition{op: "?", arg: "arg"}, `operator in "?arg" is not one of ["="]`},
		{Condition{op: "=", arg: "[a-"}, "cannot parse the argument in \"=[a-\""},
		{Condition{arg: "arg"}, "missing operator in \"arg\""},
		// OK:
		{Condition{}, "empty operator"},
		{Condition{op: "=", arg: "arg"}, ""},
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
