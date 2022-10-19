package main

import (
	_ "embed"
	"encoding/json"
	"github.com/GoogleCloudPlatform/testgrid/metadata/junit"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	//go:embed testdata/sample.xml
	sample string

	//go:embed testdata/expected.json
	expected []byte
)

func TestConstructSlackMessage(t *testing.T) {
	suites, err := junit.Parse([]byte(sample))
	assert.NoError(t, err, "If this fails, it probably indicates a problem with the sample junit report rather than the code")
	assert.NotNil(t, suites, "If this fails, it probably indicates a problem with the sample junit report rather than the code")

	blocks := convertJunitToSlack(suites)
	b, err := json.Marshal(blocks)
	assert.NoError(t, err)

	assert.Equal(t, expected, b)
}
