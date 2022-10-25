package oneofwrappers

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stretchr/testify/assert"
)

func TestOneofWrappers(t *testing.T) {
	msg := (*sensor.MsgToCompliance)(nil)

	wrappers := OneofWrappers(msg)
	assert.ElementsMatch(t,
		[]interface{}{
			(*sensor.MsgToCompliance_Config)(nil),
			(*sensor.MsgToCompliance_Trigger)(nil),
			(*sensor.MsgToCompliance_AuditLogCollectionRequest_)(nil),
		},
		wrappers)
}
