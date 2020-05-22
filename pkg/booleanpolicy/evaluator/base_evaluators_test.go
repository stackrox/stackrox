package evaluator

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFloat(t *testing.T) {
	for _, testCase := range []struct {
		f        float64
		expected string
	}{
		{-math.Pi, "-3.142"},
		{0.1241, "0.124"},
		{math.Pi, "3.142"},
		{10, "10"},
		{10.00, "10"},
		{10.000010, "10"},
		{10.100010, "10.1"},
		{100.14, "100.14"},
		{100.1444, "100.144"},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			assert.Equal(t, c.expected, formatFloat(c.f))
		})
	}
}
