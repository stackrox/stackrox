package backend

import (
	"github.com/stackrox/stackrox/pkg/auth/tokens"
)

func newSource() *sourceImpl {
	return &sourceImpl{
		revocationLayer: tokens.NewRevocationLayer(),
	}
}
