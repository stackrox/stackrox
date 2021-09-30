package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageScanCommand_Validate(t *testing.T) {
	cases := map[string]struct {
		i          imageScanCommand
		shouldFail bool
		errorMsg   string
	}{
		"valid values, no error": {
			i:          imageScanCommand{image: "nginx", format: "json"},
			shouldFail: false,
		},
		"no image value given": {
			i:          imageScanCommand{},
			shouldFail: true,
			errorMsg:   "missing image name. please specify an image name via either --image or -i",
		},
		"invalid output format given": {
			i:          imageScanCommand{image: "nginx", format: "yaml"},
			shouldFail: true,
			errorMsg:   "invalid output format given: \"yaml\". You can only specify json, csv or pretty",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.i.Validate()
			if c.shouldFail {
				require.Error(t, err)
				assert.Equal(t, c.errorMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
