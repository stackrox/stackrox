package filtered

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
)

type scopedSACFilterImpl struct {
	resourceHelper sac.ForResourceHelper
	scopeFunc      ScopeTransform
	access         storage.Access
}

func (f *scopedSACFilterImpl) Apply(ctx context.Context, from ...string) ([]string, error) {
	if ok, err := f.resourceHelper.AccessAllowed(ctx, f.access); err != nil {
		return nil, err
	} else if ok {
		return from, nil
	}

	scopeChecker := f.resourceHelper.ScopeChecker(ctx, f.access)

	errorList := errorhelpers.NewErrorList("errors during SAC filtering")
	filtered := make([]string, 0, len(from))
	for _, id := range from {
		scopes := f.scopeFunc(ctx, []byte(id))
		if len(scopes) == 0 {
			continue
		}
		ok, err := scopeChecker.AnyAllowed(ctx, scopes)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if ok {
			filtered = append(filtered, id)
		}
	}

	return filtered, errorList.ToError()
}
