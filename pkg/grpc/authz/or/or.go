package or

import (
	"context"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
)

var _ authz.Authorizer = (*or)(nil)

type or struct {
	authorizers []authz.Authorizer
}

// EnrichContext returns the input context unchanged.
func (o *or) EnrichContext(ctx context.Context, fullMethodName string) context.Context {
	newCtx := ctx
	for _, a := range o.authorizers {
		newCtx = a.EnrichContext(ctx, fullMethodName)
	}
	return newCtx
}

func (o *or) Authorized(ctx context.Context, fullMethodName string) error {
	var errors []error
	for _, a := range o.authorizers {
		err := a.Authorized(ctx, fullMethodName)
		if err == nil {
			return nil
		}
		errors = append(errors, err)
	}
	return errox.NotAuthorized.CausedBy(errorhelpers.NewErrorListWithErrors("no authorizer could authorize this request:", errors).String())
}

// Or creates an Authorizer that succeeds if any of the provided Authorizers succeed.
func Or(authorizers ...authz.Authorizer) authz.Authorizer {
	return &or{
		authorizers: authorizers,
	}
}

// SensorOr returns an Authorizer that authorizes any sensor,
// or anything that the passed authorizer authorizes.
func SensorOr(authorizer authz.Authorizer) authz.Authorizer {
	return Or(
		idcheck.SensorsOnly(),
		authorizer,
	)
}

// ScannerOr returns an Authorizer that authorizes the scanner,
// or anything that the passed authorizer authorizes.
func ScannerOr(authorizer authz.Authorizer) authz.Authorizer {
	return Or(idcheck.ScannerOnly(), authorizer)
}

// ScannerV4 returns an Authorizer that authorizes Scanner v4 Indexer or Scanner v4 Matcher.
func ScannerV4() authz.Authorizer {
	return Or(idcheck.ScannerV4IndexerOnly(), idcheck.ScannerV4MatcherOnly())
}
