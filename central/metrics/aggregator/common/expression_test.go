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
}
