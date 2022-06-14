package authn

import (
	"context"
	"errors"

	"github.com/stackrox/stackrox/pkg/auth"
	"github.com/stackrox/stackrox/pkg/contextutil"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/requestinfo"
	"github.com/stackrox/stackrox/pkg/logging"
	"gopkg.in/square/go-jose.v2/jwt"
)

var (
	log = logging.LoggerForModule()
)

type contextUpdater struct {
	extractor IdentityExtractor
}

func (u contextUpdater) updateContext(ctx context.Context) (context.Context, error) {
	id, err := u.extractor.IdentityForRequest(ctx, requestinfo.FromContext(ctx))
	if err != nil {
		if errors.Is(err, jwt.ErrExpired) {
			log.Debugf("Cannot extract identity: token expired")
		} else {
			log.Warnf("Cannot extract identity: %v", err)
		}
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
