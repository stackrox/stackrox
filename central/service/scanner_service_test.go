package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestValidateScanner(t *testing.T) {
	err := validateScanner(&v1.Scanner{
		Name:     "name",
		Type:     "type",
		Endpoint: "endpoint",
	})
	assert.NoError(t, err)

	err = validateScanner(&v1.Scanner{})
	assert.Contains(t, err.Error(), "Scanner name must be defined")
	assert.Contains(t, err.Error(), "Scanner type must be defined")
	assert.Contains(t, err.Error(), "Scanner endpoint must be defined")
}
