package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONPrinterFactory_CreatePrinter(t *testing.T) {
	cases := map[string]struct {
		j          *JSONPrinterFactory
		format     string
		shouldFail bool
		error      error
	}{
		"should not fail with valid values for factory and format": {
			j:      &JSONPrinterFactory{},
			format: "json",
		},
		"should fail with invalid format": {
			j:          &JSONPrinterFactory{},
			format:     "junit",
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := c.j.CreatePrinter(c.format)
			if c.shouldFail {
				require.Error(t, err)
				assert.Nil(t, p)
				assert.ErrorIs(t, err, c.error)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
			}
		})
	}
}
