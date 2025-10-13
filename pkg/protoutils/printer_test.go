package protoutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestWrapper(t *testing.T) {
	wrapper := NewWrapper(fixtures.GetSerializationTestAlert())
	assert.JSONEq(t, fixtures.GetJSONSerializedTestAlertWithDefaults(), wrapper.String())
}
