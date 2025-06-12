package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_expression(t *testing.T) {
	e := Expression{}
	assert.True(t, e.match("value"))

	e = Expression{&Condition{}, &Condition{}}
	assert.True(t, e.match("value"))

	e = Expression{
		&Condition{"=", "value"},
	}
	assert.True(t, e.match("value"))

	e = Expression{
		&Condition{"=", "val*"},
		&Condition{"=", "*lue"},
		&Condition{"=", "*lu*"},
		&Condition{"=", "{wrong,value}"},
	}
	assert.True(t, e.match("value"))

	e = Expression{
		&Condition{"!=", "value"},
	}
	assert.False(t, e.match("value"))

	e = Expression{
		&Condition{">", "3"},
		&Condition{"<", "5"},
	}
	assert.True(t, e.match("4"))
	assert.False(t, e.match("3"))
	assert.False(t, e.match("5"))

	e = Expression{
		&Condition{"<", "3"},
		&Condition{">", "5"},
	}
	assert.False(t, e.match("4"))
	assert.False(t, e.match("1"))
	assert.False(t, e.match("6"))

	e = Expression{
		&Condition{">", "3"},
		&Condition{"<", "5"},
		&Condition{op: "OR"},
		&Condition{">", "30"},
		&Condition{"<", "50"},
		&Condition{op: "OR"},
		&Condition{">", "300"},
		&Condition{"<", "500"},
	}

	for _, value := range []string{"4", "40", "400"} {
		assert.True(t, e.match(value))
	}
	for _, value := range []string{"3", "5", "30", "50", "300", "500"} {
		assert.False(t, e.match(value))
	}
}
