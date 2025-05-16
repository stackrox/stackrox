package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validateMetricName(t *testing.T) {
	tests := map[string]string{
		"good":             "",
		"not good":         "bad characters",
		"":                 "empty",
		"abc_defAZ0145609": "",
		"not-good":         "bad characters",
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateMetricName(name); err != nil {
				assert.Equal(t, expected, err.Error())
			} else {
				assert.Empty(t, expected)
			}
		})
	}
}
