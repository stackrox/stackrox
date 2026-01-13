package service

import "github.com/stackrox/rox/pkg/auth/tokens"

func newSource() tokens.Source {
	return &sourceImpl{
		revocationLayer: tokens.NewRevocationLayer(),
	}
}
