package filtered

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
)

type scopedSACFilterImpl struct {
	resourceHelper sac.ForResourceHelper
	scopeTransform ScopeTransform
	access         storage.Access
}

func (f *scopedSACFilterImpl) Apply(ctx context.Context, from ...string) ([]int, bool, error) {
	if ok, err := f.resourceHelper.AccessAllowed(ctx, f.access); err != nil {
		return nil, false, err
	} else if ok {
		return nil, true, nil
	}

	accessChecker := f.scopeTransform.NewCachedChecker(ctx, &f.resourceHelper, f.access)

	errorList := errorhelpers.NewErrorList("errors during SAC filtering")
	filteredIndices := make([]int, 0, len(from))
	for idx, id := range from {
		if ok, err := accessChecker.Search(ctx, id); err != nil {
			errorList.AddError(err)
			continue
		} else if ok {
			filteredIndices = append(filteredIndices, idx)
		}
	}

	return filteredIndices, false, errorList.ToError()
}
