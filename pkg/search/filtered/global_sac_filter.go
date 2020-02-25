package filtered

import (
	"context"

	"github.com/stackrox/rox/pkg/sac"
)

type globalFilterImpl struct {
	resourceHelper sac.ForResourceHelper
}

func (f *globalFilterImpl) Apply(ctx context.Context, from ...string) ([]string, error) {
	if ok, err := f.resourceHelper.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return from, nil
}
