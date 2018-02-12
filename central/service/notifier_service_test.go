package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestValidateNotifier(t *testing.T) {
	err := validateNotifier(&v1.Notifier{
		Name:       "name",
		Type:       "type",
		UiEndpoint: "endpoint",
	})
	assert.NoError(t, err)

	err = validateNotifier(&v1.Notifier{})
	assert.Contains(t, err.Error(), "Notifier name must be defined")
	assert.Contains(t, err.Error(), "Notifier type must be defined")
	assert.Contains(t, err.Error(), "Notifier UI endpoint must be defined")
}
