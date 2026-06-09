package resources

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BenchmarkPopulateImageMetadata(b *testing.B) {
	b.Setenv(features.InitContainerSupport.EnvVar(), "true")

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
			containers := makeBenchmarkContainers(tc.numContainers)
			pods := makeBenchmarkPods(tc.numContainers, tc.numPods)
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
				resetBenchmarkContainerImages(containers)
				b.StartTimer()

				wrap.populateImageMetadata(localImages, pods...)
			}
		})
	}
}

func makeBenchmarkContainers(n int) []*storage.Container {
	containers := make([]*storage.Container, n+1)
	containers[0] = &storage.Container{
		Name:  "init-setup",
		Type:  storage.ContainerType_INIT,
		Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: "registry.io/init:latest"}},
	}
	for i := range n {
		name := fmt.Sprintf("container-%d", i)
		containers[i+1] = &storage.Container{
			Name:  name,
			Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: fmt.Sprintf("registry.io/img-%d:latest", i)}},
		}
	}
	return containers
}

func makeBenchmarkPods(numContainers, numPods int) []*v1.Pod {
	pods := make([]*v1.Pod, numPods)
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
		initStatuses := []v1.ContainerStatus{{
			Name:    "init-setup",
			ImageID: "docker-pullable://registry.io/init@sha256:00000000000000000000000000000000",
		}}
		initSpecContainers := []v1.Container{{
			Name:  "init-setup",
			Image: "registry.io/init:latest",
		}}
		pods[p] = &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: time.Now().Add(-time.Duration(p) * time.Minute)}},
			Spec: v1.PodSpec{
				Containers:     specContainers,
				InitContainers: initSpecContainers,
			},
			Status: v1.PodStatus{
				ContainerStatuses:     statuses,
				InitContainerStatuses: initStatuses,
			},
		}
	}
	return pods
}

// resetBenchmarkContainerImages clears mutable fields set by populateImageMetadata
// so each benchmark iteration measures the full cold-path population logic.
func resetBenchmarkContainerImages(containers []*storage.Container) {
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
