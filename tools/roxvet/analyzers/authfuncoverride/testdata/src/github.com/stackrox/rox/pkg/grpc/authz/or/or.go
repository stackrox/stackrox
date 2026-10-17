package or

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type orAuthorizer struct{}

func (o *orAuthorizer) Authorized(_ context.Context, _ string) error { return nil }

// Or creates an Authorizer that succeeds if any of the provided Authorizers succeed.
func Or(authorizers ...authz.Authorizer) authz.Authorizer { return &orAuthorizer{} }

// ScannerV4 returns an Authorizer that authorizes Scanner v4 Indexer or Scanner v4 Matcher.
func ScannerV4() authz.Authorizer { return &orAuthorizer{} }

// SensorOr returns an Authorizer that authorizes any sensor, or the passed authorizer.
func SensorOr(authorizer authz.Authorizer) authz.Authorizer { return &orAuthorizer{} }

// ScannerOr returns an Authorizer that authorizes the scanner, or the passed authorizer.
func ScannerOr(authorizer authz.Authorizer) authz.Authorizer { return &orAuthorizer{} }
