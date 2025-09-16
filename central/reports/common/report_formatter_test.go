package common

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_replaceUnsafeRunes(t *testing.T) {
	cases := map[string]string{
		"NoSpaces":                  "NoSpaces",
		"With Spa ces":              "With_Spa_ces",
		" some!.other) chars=":      "_some__other__chars_",
		strings.Repeat("long ", 18): strings.Repeat("long_", 16),
		"":                          "",
	}
	for configName, expected := range cases {
		t.Run(configName, func(t *testing.T) {
			builder := strings.Builder{}
			replaceUnsafeRunes(&builder, configName)
			assert.Equal(t, expected, builder.String())
		})
	}
}

func Test_makeFileName(t *testing.T) {
	cases := map[string]string{
		"file name": "RHACS_Vulnerability_Report_file_name_31_December_2023.csv",
		"":          "RHACS_Vulnerability_Report_31_December_2023.csv",
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Date(2024, 1, 1, 1, 1, 1, 1, loc)

	for configName, expectedFileName := range cases {
		t.Run(configName, func(t *testing.T) {
			assert.Equal(t, expectedFileName, makeFileName(configName, today))
		})
	}
}
