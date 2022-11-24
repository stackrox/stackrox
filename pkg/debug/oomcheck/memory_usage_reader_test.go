package oomcheck

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	r := NewMemoryUsageReader()

	rImpl, ok := r.(*memoryUsageReaderImpl)
	require.True(t, ok)
	assert.Equal(t, cgroupV1StatFile, rImpl.v1StatFile)
}

func TestGetUsageCgroupV1(t *testing.T) {
	cases := map[string]struct{ limit, used uint64 }{
		"cgroupv1-gke-node":          {limit: 9223372036854771712, used: 1_445_191_680},
		"cgroupv1-gke-pod":           {limit: 2_147_483_648, used: 23_207_936},
		"cgroupv1-gke-pod-no-limits": {limit: 3_050_844_160, used: 25_706_496},
		"cgroupv1-minikube-node":     {limit: 9223372036854771712, used: 5_519_925_248},
		"cgroupv1-minikube-pod":      {limit: 2147483648, used: 61_210_624},
		"cgroupv1-ocp-node":          {limit: 9223372036854771712, used: 13_440_061_440},
		"cgroupv1-ocp-pod":           {limit: 2147483648, used: 22_138_880},
	}

	for n, expected := range cases {
		t.Run(n, func(t *testing.T) {
			reader := newWithDirectory(path.Join("testfiles/real-fs", n))

			usage, err := reader.GetUsage()

			assert.NoError(t, err)
			assert.Equal(t, expected.used, usage.Used)
			assert.Equal(t, expected.limit, usage.Limit)
		})
	}
}
