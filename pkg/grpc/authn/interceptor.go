package authn

import (
	"context"

	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

type contextUpdater struct {
	extractor IdentityExtractor
}

func (u contextUpdater) updateContext(ctx context.Context) (context.Context, error) {
	id := u.extractor.IdentityForRequest(requestinfo.FromContext(ctx))
	if id == nil {
		return ctx, nil
	}
	return context.WithValue(ctx, identityContextKey{}, id), nil
}

// ContextUpdater returns a context updater for the given identity extractors
func ContextUpdater(extractors ...IdentityExtractor) contextutil.ContextUpdater {
	return contextUpdater{extractor: CombineExtractors(extractors...)}.updateContext
}
