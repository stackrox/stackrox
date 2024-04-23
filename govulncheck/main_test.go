package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectAffectedVulnerabilities_WithNoExceptions(t *testing.T) {
	output, err := collectAffectedVulnerabilities(
		"testdata/vulns", &ExceptionConfig{Exceptions: map[string]*Exception{}})
	assert.NoError(t, err)

	assert.Len(t, output.Data, 1)
	osvEntry := output.Data[0]
	assert.Equal(t, "GO-2024-2687", osvEntry["id"])
	assert.Equal(t, "HTTP/2 CONTINUATION flood in net/http", osvEntry["summary"])
}

func TestCollectAffectedVulnerabilities_WithException(t *testing.T) {
	output, err := collectAffectedVulnerabilities(
		"testdata/vulns", &ExceptionConfig{Exceptions: map[string]*Exception{
			"GO-2024-2687": {Reason: "Testing"},
		}})
	assert.NoError(t, err)
	assert.Len(t, output.Data, 0)
}
