package expiry

import (
	"context"
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
		func(ctx context.Context, _ tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, s)
		},
	)
}

func track(ctx context.Context, s service.Service) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		if s == nil {
			return
		}
		var f finding
		for i, component := range v1.GetCertExpiry_Component_name {
			if v1.GetCertExpiry_Component(i) == v1.GetCertExpiry_UNKNOWN {
				continue
			}
			result, err := s.GetCertExpiry(ctx, &v1.GetCertExpiry_Request{
				Component: v1.GetCertExpiry_Component(i),
			})
			f.component = component
			if result != nil {
				f.hoursUntilExpiration = int(time.Until(result.GetExpiry().AsTime()).Hours())
			}
			if !yield(&f, err) {
				return
			}
		}
	}
}
