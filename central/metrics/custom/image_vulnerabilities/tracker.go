package image_vulnerabilities

import (
	"context"
	"iter"

	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(ds deploymentDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"image_vuln",
		"image vulnerabilities",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds deploymentDS.DataStore) iter.Seq[*finding] {
	return func(yield func(*finding) bool) {
		var f finding
		f.err = ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
			f.deployment = deployment
			images, err := ds.GetImagesForDeployment(ctx, deployment)
			if err != nil {
				return err
			}
			for _, f.image = range images {
				if !forEachImageVuln(yield, &f) {
					return tracker.ErrStopIterator
				}
			}
			return nil
		})
		// Report walking error.
		if f.err != nil && !errors.Is(f.err, tracker.ErrStopIterator) {
			yield(&f)
		}
	}
}

// forEachImageVuln yields a finding for every vulnerability associated with
// each image name.
func forEachImageVuln(yield func(*finding) bool, f *finding) bool {
	for _, f.component = range f.image.GetScan().GetComponents() {
		for _, f.vuln = range f.component.GetVulns() {
			for _, f.name = range f.image.GetNames() {
				if !yield(f) {
					return false
				}
			}
		}
	}
	return true
}
