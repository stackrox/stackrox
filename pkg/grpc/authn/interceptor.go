package authn

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
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
		return context.WithValue(ctx, identityErrorContextKey{}, errox.NewErrNoCredentials(err.Error())), nil
	}
	if id != nil {
		// Only service identities can have no roles assigned.
		if len(id.Roles()) == 0 && id.Service() == nil {
			return context.WithValue(ctx, identityErrorContextKey{}, errox.GenericNoValidRole()), nil
		}
		return context.WithValue(ctx, identityContextKey{}, id), nil
	}
	return ctx, nil
}

// ContextUpdater returns a context updater for the given identity extractors
func ContextUpdater(extractors ...IdentityExtractor) contextutil.ContextUpdater {
	return contextUpdater{extractor: CombineExtractors(extractors...)}.updateContext
}
