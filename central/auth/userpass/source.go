package userpass

import (
	"errors"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

const (
	id = `https://stackrox.io/jwt-sources#username-password`
)

type source struct{}

func (s *source) Validate(claims *tokens.Claims) error {
	if claims.ID == "" {
		return errors.New("token has no ID")
	}
	return nil
}

func (s *source) ID() string {
	return id
}
