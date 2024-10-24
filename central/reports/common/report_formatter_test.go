package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeSafeFileName(t *testing.T) {
	cases := map[string]string{
		"NoSpaces":                  "NoSpaces",
		"With Spa ces":              "With_Spa_ces",
		" some!.other) chars=":      "_some__other__chars_",
		strings.Repeat("long ", 18): strings.Repeat("long_", 16),
	}
	for configName, expectedFileName := range cases {
		t.Run(configName, func(t *testing.T) {
			assert.Equal(t, expectedFileName, makeSafeFileName(configName))
		})
	}
}
