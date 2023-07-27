package injector

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_average(t *testing.T) {
	a := average()
	require.NotNil(t, a)
	assert.Equal(t, int32(0), a.NumNodes)
	assert.Equal(t, int32(0), a.NumCpuUnits)

	metrics := []*storage.Usage{{
		NumNodes:    0,
		NumCpuUnits: 100,
	}, {
		NumNodes:    10,
		NumCpuUnits: 0,
	}}
	a = average(metrics...)
	require.NotNil(t, a)
	assert.Equal(t, int32(5), a.NumNodes)
	assert.Equal(t, int32(50), a.NumCpuUnits)
}
