package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaybeTrimPrefix(t *testing.T) {
	cases := []struct {
		name            string
		s               string
		p               string
		expectedString  string
		expectedPresent bool
	}{
		{
			name:            "HasPrefix",
			s:               "Prefix...",
			p:               "Prefix",
			expectedString:  "...",
			expectedPresent: true,
		},
		{
			name:            "DoesNotHavePrefix",
			s:               "Prefix...",
			p:               "xxy",
			expectedString:  "Prefix...",
			expectedPresent: false,
		},
		{
			name:            "Tab",
			s:               "\t",
			p:               "\t",
			expectedString:  "",
			expectedPresent: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, present := MaybeTrimPrefix(c.s, c.p)
			assert.Equal(t, c.expectedString, out)
			assert.Equal(t, c.expectedPresent, present)
		})
	}
}
