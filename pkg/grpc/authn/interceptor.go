package authn

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/contextutil"
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
		return context.WithValue(ctx, identityErrorContextKey{}, err), nil
	}
	if id == nil {
		return ctx, nil
	}
	return context.WithValue(ctx, identityContextKey{}, id), nil
}

// ContextUpdater returns a context updater for the given identity extractors
func ContextUpdater(extractors ...IdentityExtractor) contextutil.ContextUpdater {
	return contextUpdater{extractor: CombineExtractors(extractors...)}.updateContext
}
