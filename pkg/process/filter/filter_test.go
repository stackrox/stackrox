package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestBasicFilterFunctions(t *testing.T) {
	filter := NewFilter(2, 2, []int{3, 2, 1})

	pi := fixtures.GetProcessIndicator()
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	filter.Delete(pi.GetDeploymentId())

	assert.True(t, filter.Add(pi))
}

func TestBasicFilter(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		expected []bool
	}{
		{
			name:     "general stopping",
			args:     []string{"1 2 3", "1 2 3", "1 2 3"},
			expected: []bool{true, true, false},
		},
		{
			name:     "general long stopping",
			args:     []string{"1 2 3 4 5", "1 2 3 4 5", "1 2 3 4 5"},
			expected: []bool{true, true, false},
		},
		{
			name:     "general short stopping",
			args:     []string{"1", "1", "1"},
			expected: []bool{true, true, false},
		},
		{
			name: "fan out check",
			// Fan out applies to process first then args
			// "process" = fan out of 3
			// "1" = fan out of 2
			// "2" = fan out of 1
			args:     []string{"1 2 3", "1 2 3", "1 2 2"},
			expected: []bool{true, true, false},
		},
		{
			name:     "varying fan out",
			args:     []string{"1", "1 2", "1 2 3", "1 2 4"},
			expected: []bool{true, true, true, false},
		},
		{
			name:     "high fan out in first level",
			args:     []string{"1", "2", "3", "4"},
			expected: []bool{true, true, true, false},
		},
		{
			name:     "verify failed filters dont impact fan out",
			args:     []string{"1", "1", "1", "1 2 3"},
			expected: []bool{true, true, false, true},
		},
	}

	for _, c := range cases {
		currCase := c
		t.Run(currCase.name, func(t *testing.T) {
			filter := NewFilter(2, 2, []int{3, 2, 1})
			for i, arg := range currCase.args {
				result := filter.Add(&storage.ProcessIndicator{
					PodId:         "pod",
					ContainerName: "name",
					Signal: &storage.ProcessSignal{
						ContainerId:  "id",
						ExecFilePath: "path",
						Args:         arg,
					},
				})
				assert.Equal(t, currCase.expected[i], result)
			}
		})
	}
}

func TestMultiProcessFilter(t *testing.T) {
	filter := NewFilter(2, 2, []int{3, 2, 1})

	// Ensure that different (pod, container name) pairs are isolated
	pi := fixtures.GetProcessIndicator()
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	pi.Signal.ContainerId = "newcontainer"
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))
}

func TestMaxFilePaths(t *testing.T) {
	filter := NewFilter(2, 2, []int{3, 2, 1})

	// Ensure that different (pod, container name) pairs are isolated
	pi := fixtures.GetProcessIndicator()
	assert.True(t, filter.Add(pi))

	pi.Signal.Name = "signal2"
	assert.True(t, filter.Add(pi))

	pi.Signal.Name = "signal3"
	assert.False(t, filter.Add(pi))
}

func TestPodUpdate(t *testing.T) {
	filter := NewFilter(2, 2, []int{3, 2, 1}).(*filterImpl)

	pi := fixtures.GetProcessIndicator()
	filter.Add(pi)

	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 1)

	pod := fixtures.GetPod()
	assert.Equal(t, pod.GetDeploymentId(), pi.GetDeploymentId())

	filter.UpdateByPod(pod)
	// The container id of the process and the pod match so there should be no change
	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 1)

	// The container id has changed so the container reference should be removed, but the deployment reference should remain
	filter.Add(pi)
	pod.LiveInstances[0].InstanceId.Id = "newcontainerid"
	filter.UpdateByPod(pod)
	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 0)
}

func TestPodUpdateWithInstanceTruncation(t *testing.T) {
	filter := NewFilter(2, 2, []int{3, 2, 1}).(*filterImpl)

	pi := fixtures.GetProcessIndicator()
	pi.Signal.ContainerId = "0123456789ab"
	filter.Add(pi)

	pod := fixtures.GetPod()
	// instance id to > 12 digits but it should be truncated to the proper length
	pod.LiveInstances[0].InstanceId.Id = "0123456789abcdef"
	filter.UpdateByPod(pod)
	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 1)
}

func TestFilterWithEmptyFanOut(t *testing.T) {
	// Empty fanOut array means only track unique processes, no argument tracking
	// All calls to the same process share the same hit counter regardless of args
	filter := NewFilter(2, 5, []int{})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
			Args:         "arg1",
		},
	}

	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1 arg2"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg2"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg3"
	assert.False(t, filter.Add(pi))

	// But a different process should still work
	pi.Signal.ExecFilePath = "different_path"
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))
}

func TestFilterWithEmptyFanOutMaxExactPathMatches1(t *testing.T) {
	// Empty fanOut array means only track unique processes, no argument tracking
	// All calls to the same process share the same hit counter regardless of args
	filter := NewFilter(1, 5, []int{})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
		},
	}

	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1 arg2"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg2"
	assert.False(t, filter.Add(pi))

	// But a different process should still work
	pi.Signal.ExecFilePath = "different_path"
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))
}

func TestFilterWithSingleLevelFanOut1(t *testing.T) {
	// Single-level fanOut with limit of 2
	filter := NewFilter(2, 5, []int{1})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
			Args:         "arg1",
		},
	}

	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1 arg2"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg2"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg3"
	assert.False(t, filter.Add(pi))

	// But a different process should still work
	pi.Signal.ExecFilePath = "different_path"
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))
}

func TestFilterWithTwoLevelFanOut(t *testing.T) {
	// Single-level fanOut with limit of 2
	filter := NewFilter(9, 5, []int{3, 2})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
			Args:         "arg1a arg2a",
		},
	}

	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg1b arg2a"
	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg1c arg2a"
	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg1d arg2a"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1a arg2b"
	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg1a arg2c"
	assert.False(t, filter.Add(pi))
}

func TestFilterWithSingleLevelFanOut1MaxExactPathMatches1(t *testing.T) {
	filter := NewFilter(1, 5, []int{1})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
		},
	}

	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1"
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg1 arg2"
	assert.False(t, filter.Add(pi))

	pi.Signal.Args = "arg2"
	assert.False(t, filter.Add(pi))

	// But a different process should still work
	pi.Signal.ExecFilePath = "different_path"
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))
}

func TestFilterWithSingleLevelFanOut(t *testing.T) {
	// Single-level fanOut with limit of 2
	filter := NewFilter(2, 5, []int{2})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
			Args:         "arg1",
		},
	}

	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg1 arg2"
	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg2"
	assert.True(t, filter.Add(pi))

	pi.Signal.Args = "arg3"
	assert.False(t, filter.Add(pi))

	// But a different process should still work
	pi.Signal.ExecFilePath = "different_path"
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))
}

func TestFilterWithManyLevels(t *testing.T) {
	// Many-level fanOut - note that deeper levels have smaller fanOut
	// This tests that the filter can handle many levels correctly
	filter := NewFilter(2, 5, []int{10, 8, 6, 4, 2, 2, 2})

	pi := &storage.ProcessIndicator{
		PodId:         "pod",
		ContainerName: "name",
		DeploymentId:  "deployment",
		Signal: &storage.ProcessSignal{
			ContainerId:  "id",
			ExecFilePath: "path",
			Args:         "a b c d e f g",
		},
	}

	// Should handle deep argument trees
	assert.True(t, filter.Add(pi))
	assert.True(t, filter.Add(pi))
	assert.False(t, filter.Add(pi))

	// Different deep path should work because we have fanOut[6] = 2
	pi.Signal.Args = "a b c d e f h"
	assert.True(t, filter.Add(pi))
}

// TestEmptyFanOutVsFanOut1 demonstrates the key difference between empty fanOut and fanOut [1]
func TestEmptyFanOutVsFanOut1(t *testing.T) {
	t.Run("empty fanOut - all args share same counter", func(t *testing.T) {
		// Empty fanOut: no argument tracking at all
		// All invocations of the same process share one hit counter regardless of arguments
		filter := NewFilter(3, 5, []int{})

		pi1 := &storage.ProcessIndicator{
			PodId:         "pod",
			ContainerName: "name",
			DeploymentId:  "deployment",
			Signal: &storage.ProcessSignal{
				ContainerId:  "id",
				ExecFilePath: "bash",
			},
		}

		pi2 := &storage.ProcessIndicator{
			PodId:         "pod",
			ContainerName: "name",
			DeploymentId:  "deployment",
			Signal: &storage.ProcessSignal{
				ContainerId:  "id",
				ExecFilePath: "bash",
				Args:         "arg1",
			},
		}

		assert.True(t, filter.Add(pi1))
		assert.True(t, filter.Add(pi2))
		assert.True(t, filter.Add(pi1))
		assert.False(t, filter.Add(pi2))
		assert.False(t, filter.Add(pi1))
		assert.False(t, filter.Add(pi2))
		assert.False(t, filter.Add(pi1))
		assert.False(t, filter.Add(pi2))

		pi1.Signal.Args = "arg2"
		assert.False(t, filter.Add(pi1))

		pi1.Signal.Args = "completely different args"
		assert.False(t, filter.Add(pi1))

		// Different process works fine
		pi1.Signal.ExecFilePath = "python"
		pi1.Signal.Args = "script.py"
		assert.True(t, filter.Add(pi1)) // New process, new counter
	})

	t.Run("fanOut [1] - only one first-arg allowed, separate counters per arg", func(t *testing.T) {
		// FanOut [1]: tracks first argument level with fanOut limit of 1
		// Only 1 unique first argument allowed, but all paths starting with that arg share a counter
		// (because fanOut only has 1 level - no second level to differentiate further)
		filter := NewFilter(3, 5, []int{1})

		pi1 := &storage.ProcessIndicator{
			PodId:         "pod",
			ContainerName: "name",
			DeploymentId:  "deployment",
			Signal: &storage.ProcessSignal{
				ContainerId:  "id",
				ExecFilePath: "bash",
			},
		}

		pi2 := &storage.ProcessIndicator{
			PodId:         "pod",
			ContainerName: "name",
			DeploymentId:  "deployment",
			Signal: &storage.ProcessSignal{
				ContainerId:  "id",
				ExecFilePath: "bash",
				Args:         "arg1",
			},
		}

		// This is different from the previous test where only the first three process are accepted.
		assert.True(t, filter.Add(pi1))
		assert.True(t, filter.Add(pi2))
		assert.True(t, filter.Add(pi1))
		assert.True(t, filter.Add(pi2))
		assert.True(t, filter.Add(pi1))
		assert.True(t, filter.Add(pi2))
		assert.False(t, filter.Add(pi1))
		assert.False(t, filter.Add(pi2))

		pi1.Signal.Args = "arg2"
		assert.False(t, filter.Add(pi1))

		pi1.Signal.Args = "completely different args"
		assert.False(t, filter.Add(pi1))

		// Different process works fine
		pi1.Signal.ExecFilePath = "python"
		pi1.Signal.Args = "script.py"
		assert.True(t, filter.Add(pi1)) // New process, new counter
	})
}

// TestFanOut1VsFanOut1_2 shows the difference between [1] and [1, 2]
func TestFanOut1VsFanOut1_2(t *testing.T) {
	t.Run("fanOut [1] - only tracks first arg, all variations share counter", func(t *testing.T) {
		filter := NewFilter(3, 5, []int{1})

		pi := &storage.ProcessIndicator{
			PodId:         "pod",
			ContainerName: "name",
			DeploymentId:  "deployment",
			Signal: &storage.ProcessSignal{
				ContainerId:  "id",
				ExecFilePath: "bash",
				Args:         "arg1",
			},
		}

		// "arg1" alone
		assert.True(t, filter.Add(pi))  // hit 1
		assert.True(t, filter.Add(pi))  // hit 2
		assert.True(t, filter.Add(pi))  // hit 3
		assert.False(t, filter.Add(pi)) // filtered

		// "arg1 sub1" - shares counter with "arg1" alone
		pi.Signal.Args = "arg1 sub1"
		assert.False(t, filter.Add(pi)) // filtered - same counter
	})

	t.Run("fanOut [1, 2] - tracks two levels, separate counters", func(t *testing.T) {
		filter := NewFilter(3, 5, []int{1, 2})

		pi := &storage.ProcessIndicator{
			PodId:         "pod",
			ContainerName: "name",
			DeploymentId:  "deployment",
			Signal: &storage.ProcessSignal{
				ContainerId:  "id",
				ExecFilePath: "bash",
				Args:         "arg1",
			},
		}

		// "arg1" alone - creates its own path
		assert.True(t, filter.Add(pi))  // hit 1 on "arg1" path (no second arg)
		assert.True(t, filter.Add(pi))  // hit 2
		assert.True(t, filter.Add(pi))  // hit 3
		assert.False(t, filter.Add(pi)) // filtered

		// "arg1 sub1" - DIFFERENT path! Has second-level child "sub1"
		pi.Signal.Args = "arg1 sub1"
		assert.True(t, filter.Add(pi))  // hit 1 on "arg1 sub1" path (NEW counter!)
		assert.True(t, filter.Add(pi))  // hit 2
		assert.True(t, filter.Add(pi))  // hit 3
		assert.False(t, filter.Add(pi)) // filtered

		// "arg1 sub2" - Another DIFFERENT path! Has second-level child "sub2"
		pi.Signal.Args = "arg1 sub2"
		assert.True(t, filter.Add(pi)) // hit 1 on "arg1 sub2" path (NEW counter!)
		assert.True(t, filter.Add(pi)) // hit 2

		// "arg1 sub3" - REJECTED: fanOut[1] = 2, already have "sub1" and "sub2"
		pi.Signal.Args = "arg1 sub3"
		assert.False(t, filter.Add(pi)) // fanOut limit reached at second level
	})
}
