package resources

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/set"
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
