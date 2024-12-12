package pgsearch

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type PredicatePair struct {
	value             interface{}
	expectedSelection bool
}

func TestCIDRQuery(t *testing.T) {
	_, valueCIDR, _ := net.ParseCIDR("230.127.112.0/24")

	cases := []struct {
		value             string
		expectErr         bool
		expectedQuery     string
		expectedValues    []interface{}
		goEquivalentPairs *[]PredicatePair
	}{
		{
			value:          valueCIDR.String(),
			expectedQuery:  "blah <<= $$",
			expectedValues: []interface{}{valueCIDR},
			goEquivalentPairs: &[]PredicatePair{
				{
					value:             "1.1.1.1/32",
					expectedSelection: false,
				},
				{
					value:             "230.127.112.8/32",
					expectedSelection: true,
				},
				{
					value:             "230.127.112.8/24",
					expectedSelection: true,
				},
				{
					value:             "230.127.112.128/25",
					expectedSelection: true,
				},
				{
					value:             "230.127.112.0/23",
					expectedSelection: false,
				},
				{
					value:             true, // unexpected type
					expectedSelection: false,
				},
				{
					value:             "invalid",
					expectedSelection: false,
				},
			},
		},
		{
			value:     "230.127.112.0",
			expectErr: true,
		},
		{
			value:     "",
			expectErr: true,
		},
		{
			value:     "invalid",
			expectErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			actual, err := newCIDRQuery(&queryAndFieldContext{
				qualifiedColumnName: "blah",
				value:               c.value,
			})
			if c.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.expectedQuery, actual.Where.Query)
			assert.Equal(t, c.expectedValues, actual.Where.Values)
			for _, p := range *c.goEquivalentPairs {
				assert.Equal(t, p.expectedSelection, actual.Where.equivalentGoFunc(p.value), p.value)
			}
		})
	}

	assert.False(t, IPNetContainsSubnet(nil, ""))
}
