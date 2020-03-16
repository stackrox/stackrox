package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestBasicFilterFunctions(t *testing.T) {
	filter := NewFilter(2, []int{3, 2, 1})

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
			filter := NewFilter(2, []int{3, 2, 1})
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
	filter := NewFilter(2, []int{3, 2, 1})

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

func TestDeploymentUpdate(t *testing.T) {
	filter := NewFilter(2, []int{3, 2, 1}).(*filterImpl)

	pi := fixtures.GetProcessIndicator()
	filter.Add(pi)

	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 1)

	dep := fixtures.GetDeployment()
	assert.Equal(t, dep.GetId(), pi.GetDeploymentId())

	filter.Update(dep)
	// The container id of the process and the deployment match so there should be no change
	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 1)

	// The container id has changed so the container reference should be removed, but the deployment reference should remain
	filter.Add(pi)
	dep.Containers[0].Instances[0].InstanceId.Id = "newcontainerid"
	filter.Update(dep)
	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 0)
}

func TestPodUpdate(t *testing.T) {
	filter := NewFilter(2, []int{3, 2, 1}).(*filterImpl)

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
	pod.Instances[0].InstanceId.Id = "newcontainerid"
	filter.UpdateByPod(pod)
	assert.Len(t, filter.containersInDeployment, 1)
	assert.Len(t, filter.containersInDeployment[pi.GetDeploymentId()], 0)
}
