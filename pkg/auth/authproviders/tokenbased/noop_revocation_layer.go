package tokenbased

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

var _ tokens.RevocationLayer = (*noopRevocationLayer)(nil)

type noopRevocationLayer struct{}

func (*noopRevocationLayer) Validate(_ context.Context, _ *tokens.Claims) error { return nil }

func (*noopRevocationLayer) Revoke(_ string, _ time.Time) {}

func (*noopRevocationLayer) IsRevoked(_ string) bool { return false }
