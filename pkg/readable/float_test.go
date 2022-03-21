package readable

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloat(t *testing.T) {
	for _, testCase := range []struct {
		f           float64
		maxPlaces   int
		expectedOut string
	}{
		{3.1253, 3, "3.125"},
		{3.1, 3, "3.1"},
		{3.12, 3, "3.12"},
		{3.123, 3, "3.123"},
		{3.1234, 3, "3.123"},
		{3.1234, 0, "3"},

		{3.1234, -1, "3.1234"},
		{-math.Pi, 3, "-3.142"},
		{0.1241, 3, "0.124"},
		{math.Pi, 3, "3.142"},
		{10, 3, "10"},
		{10.00, 3, "10"},
		{10.000010, 3, "10"},
		{10.100010, 3, "10.1"},
		{100.14, 3, "100.14"},
		{100.1444, 3, "100.144"},
	} {
		t.Run(fmt.Sprintf("%+v", testCase), func(t *testing.T) {
			assert.Equal(t, testCase.expectedOut, Float(testCase.f, testCase.maxPlaces))
		})
	}
}
