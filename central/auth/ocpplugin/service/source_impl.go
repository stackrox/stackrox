package service

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

const (
	id = "https://stackrox.io/jwt-sources#ocp-rox-tokens"
)

var (
	_ tokens.Source = (*sourceImpl)(nil)
)

type sourceImpl struct {
	revocationLayer tokens.RevocationLayer
}

func (s *sourceImpl) ID() string {
	return id
}

func (s *sourceImpl) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

func (s *sourceImpl) Revoke(tokenID string, expiry time.Time) {
	s.revocationLayer.Revoke(tokenID, expiry)
}

func (s *sourceImpl) IsRevoked(tokenID string) bool {
	return s.revocationLayer.IsRevoked(tokenID)
}
