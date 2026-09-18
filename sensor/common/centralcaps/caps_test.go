package centralcaps

import (
	"testing"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stretchr/testify/assert"
)

func TestReceived(t *testing.T) {
	t.Cleanup(Reset)
	Reset()

	assert.False(t, Received())
	assert.False(t, Has(centralsensor.VirtualMachinesSupported))

	Set(nil)
	assert.True(t, Received())
	assert.False(t, Has(centralsensor.VirtualMachinesSupported))

	Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
	assert.True(t, Received())
	assert.True(t, Has(centralsensor.VirtualMachinesSupported))

	Reset()
	assert.False(t, Received())
	assert.False(t, Has(centralsensor.VirtualMachinesSupported))
}
