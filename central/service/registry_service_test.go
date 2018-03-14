package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestValidateRegistry(t *testing.T) {
	err := validateRegistry(&v1.Registry{
		Name:     "name",
		Type:     "type",
		Endpoint: "endpoint",
	})
	assert.NoError(t, err)

	err = validateRegistry(&v1.Registry{})
	assert.Contains(t, err.Error(), "Registry name must be defined")
	assert.Contains(t, err.Error(), "Registry type must be defined")
	assert.Contains(t, err.Error(), "Registry endpoint must be defined")
}
