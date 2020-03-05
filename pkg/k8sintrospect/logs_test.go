package k8sintrospect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateLogData(t *testing.T) {
	cases := map[string]struct {
		InputLogData                       string
		MaxLogFileSize, MaxFirstLineCutOff int

		ExpectedTruncatedData string
	}{
		"do not truncate": {
			InputLogData:          "foo\nbar\nbaz\n",
			MaxLogFileSize:        12,
			MaxFirstLineCutOff:    0,
			ExpectedTruncatedData: "foo\nbar\nbaz\n",
		},
		"truncate in middle of line with first line cutoff": {
			InputLogData:          "foo\nbar\nbaz\n",
			MaxLogFileSize:        10,
			MaxFirstLineCutOff:    4,
			ExpectedTruncatedData: "bar\nbaz\n",
		},
		"truncate in middle of line without first line cutoff": {
			InputLogData:          "foo\nbar\nbaz\n",
			MaxLogFileSize:        10,
			MaxFirstLineCutOff:    0,
			ExpectedTruncatedData: "o\nbar\nbaz\n",
		},
		"truncate at line boundary with theoretical first line cutoff": {
			InputLogData:          "foo\nbar\nbaz\n",
			MaxLogFileSize:        8,
			MaxFirstLineCutOff:    4,
			ExpectedTruncatedData: "bar\nbaz\n",
		},
	}

	for desc, c := range cases {
		t.Run(desc, func(t *testing.T) {
			rawLogData := []byte(c.InputLogData)
			truncatedLogData := string(truncateLogData(rawLogData, c.MaxLogFileSize, c.MaxFirstLineCutOff))
			assert.Equal(t, c.ExpectedTruncatedData, truncatedLogData)
		})
	}
}
