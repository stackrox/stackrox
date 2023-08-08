package injector

import (
	"testing"

	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_average(t *testing.T) {
	a := average()
	require.NotNil(t, a)
	assert.Equal(t, int64(0), a.GetNumNodes())
	assert.Equal(t, int64(0), a.GetNumCPUUnits())

	metrics := []datastore.Data{
		&datastore.DataImpl{
			NumNodes:    0,
			NumCpuUnits: 100,
		}, &datastore.DataImpl{
			NumNodes:    10,
			NumCpuUnits: 0,
		}}
	a = average(metrics...)
	require.NotNil(t, a)
	assert.Equal(t, int64(5), a.GetNumNodes())
	assert.Equal(t, int64(50), a.GetNumCPUUnits())
}
