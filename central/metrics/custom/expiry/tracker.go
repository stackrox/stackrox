package expiry

import (
	"context"
	"iter"
	"time"

	"github.com/stackrox/rox/central/credentialexpiry/service"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

func New(s service.Service) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"cert_exp",
		"certificate expiry",
		LazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return track(ctx, s)
		},
	)
}

func track(ctx context.Context, s service.Service) iter.Seq[*finding] {
	return func(yield func(*finding) bool) {
		if s == nil {
			return
		}
		var f finding
		for i, component := range v1.GetCertExpiry_Component_name {
			if v1.GetCertExpiry_Component(i) == v1.GetCertExpiry_UNKNOWN {
				continue
			}
			gr := &v1.GetCertExpiry_Request{}
			gr.SetComponent(v1.GetCertExpiry_Component(i))
			result, err := s.GetCertExpiry(ctx, gr)
			f.component = component
			f.err = err
			if result != nil {
				f.hoursUntilExpiration = int(time.Until(result.GetExpiry().AsTime()).Hours())
			}
			if !yield(&f) {
				return
			}
		}
	}
}
