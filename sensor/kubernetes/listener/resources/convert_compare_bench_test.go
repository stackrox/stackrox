package resources

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarkPopulateImageMetadataCompare_SteadyStateSortedPods measures the
// steady-state cost of populateImageMetadata once the pod slice is already in
// the expected newest-to-oldest order. This intentionally models the warm path
// after an earlier call has sorted the shared pod fixture, and it excludes the
// container image reset from timing so the benchmark isolates the metadata
// population work rather than fixture repair.
func BenchmarkPopulateImageMetadataCompare_SteadyStateSortedPods(b *testing.B) {
	cases := []struct {
		numContainers int
		numPods       int
	}{
		{1, 1},
		{5, 3},
		{20, 10},
	}

	for _, tc := range cases {
		b.Run(fmt.Sprintf("%dc_%dp", tc.numContainers, tc.numPods), func(b *testing.B) {
			containers := makeCompareBenchmarkContainers(tc.numContainers)
			pods := makeCompareBenchmarkPods(tc.numContainers, tc.numPods)
			wrap := &deploymentWrap{
				Deployment: &storage.Deployment{
					Name:       "bench-deploy",
					Namespace:  "default",
					Containers: containers,
				},
			}
			localImages := set.NewStringSet()

			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				resetCompareBenchmarkContainerImages(containers)
				b.StartTimer()
				wrap.populateImageMetadata(localImages, pods...)
			}
		})
	}
}

func makeCompareBenchmarkContainers(n int) []*storage.Container {
	containers := make([]*storage.Container, n)
	for i := range n {
		name := fmt.Sprintf("container-%d", i)
		containers[i] = &storage.Container{
			Name:  name,
			Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: fmt.Sprintf("registry.io/img-%d:latest", i)}},
		}
	}
	return containers
}

func makeCompareBenchmarkPods(numContainers, numPods int) []*v1.Pod {
	pods := make([]*v1.Pod, numPods)
	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for p := range numPods {
		statuses := make([]v1.ContainerStatus, numContainers)
		specContainers := make([]v1.Container, numContainers)
		for i := range numContainers {
			name := fmt.Sprintf("container-%d", i)
			statuses[i] = v1.ContainerStatus{
				Name:    name,
				ImageID: fmt.Sprintf("docker-pullable://registry.io/img-%d@sha256:%032x", i, i),
			}
			specContainers[i] = v1.Container{
				Name:  name,
				Image: fmt.Sprintf("registry.io/img-%d:latest", i),
			}
		}
		pods[p] = &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: baseTime.Add(-time.Duration(p) * time.Minute)}},
			Spec:       v1.PodSpec{Containers: specContainers},
			Status:     v1.PodStatus{ContainerStatuses: statuses},
		}
	}
	return pods
}

func resetCompareBenchmarkContainerImages(containers []*storage.Container) {
	for _, c := range containers {
		img := c.GetImage()
		if img == nil {
			continue
		}
		img.Id = ""
		img.IdV2 = ""
		img.NotPullable = false
		img.IsClusterLocal = false
	}
}
