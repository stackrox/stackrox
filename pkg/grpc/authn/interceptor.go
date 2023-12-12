package authn

import (
	"context"

	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type contextUpdater struct {
	extractor IdentityExtractor
}

func (u contextUpdater) updateContext(ctx context.Context) (context.Context, error) {
	ri := requestinfo.FromContext(ctx)
	id, err := u.extractor.IdentityForRequest(ctx, ri)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(ri.Hostname, "Cannot extract identity: %v", err)

		// Ignore id value if error is not nil.
		return context.WithValue(ctx, identityErrorContextKey{}, errox.NoCredentials.CausedBy(err)), nil
	}
	if id != nil {
		// Only service identities can have no roles assigned.
		if len(id.Roles()) == 0 && id.Service() == nil {
			return context.WithValue(ctx, identityErrorContextKey{}, auth.ErrNoValidRole), nil
		}
		return context.WithValue(ctx, identityContextKey{}, id), nil
	}
	return ctx, nil
}

// ContextUpdater returns a context updater for the given identity extractors
func ContextUpdater(extractors ...IdentityExtractor) contextutil.ContextUpdater {
	return contextUpdater{extractor: CombineExtractors(extractors...)}.updateContext
}
