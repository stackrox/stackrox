package k8s

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestConvertQuantityToResource(t *testing.T) {
	a := assert.New(t)

	coreTestCases := []float32{0.1, 0.5, 1, 1.5, 2, 3.151}
	for _, core := range coreTestCases {
		a.Equal(core, ConvertQuantityToCores(ConvertCoresToQuantity(core)))
	}

	mbTestCases := []float32{100, 200, 300.5, 400.52124}
	for _, mb := range mbTestCases {
		a.Equal(mb, ConvertQuantityToMB(ConvertMBToQuantity(mb)))
	}
}

func TestConvertQuantityToCores(t *testing.T) {
	cases := []struct {
		quantity resource.Quantity
		expected float32
	}{
		{
			quantity: resource.MustParse("20m"),
			expected: 0.02,
		},
		{
			quantity: resource.MustParse("200m"),
			expected: 0.2,
		},
		{
			quantity: resource.MustParse("2"),
			expected: 2.0,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.quantity.String(), func(t *testing.T) {
			assert.Equal(t, c.expected, ConvertQuantityToCores(&c.quantity))
		})
	}
}

func TestConvertQuantityToMb(t *testing.T) {
	cases := []struct {
		quantity resource.Quantity
		expected float32
	}{
		{
			quantity: resource.MustParse("128974848"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("129e6"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("129M"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("123Mi"),
			expected: 123,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.quantity.String(), func(t *testing.T) {
			assert.True(t, math.Abs(float64(c.expected-ConvertQuantityToMB(&c.quantity))) < 0.1)
		})
	}
}
