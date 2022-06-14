package filtered

import (
	"context"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
)

type globalFilterImpl struct {
	resourceHelper sac.ForResourceHelper
	access         storage.Access
}

func (f *globalFilterImpl) Apply(ctx context.Context, from ...string) ([]int, bool, error) {
	if ok, err := f.resourceHelper.AccessAllowed(ctx, f.access); err != nil || !ok {
		return nil, false, err
	}
	return nil, true, nil
}
