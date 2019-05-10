package backend

import (
	"github.com/stackrox/rox/pkg/auth/tokens"
)

func newSource() *sourceImpl {
	return &sourceImpl{
		revocationLayer: tokens.NewRevocationLayer(),
	}
}
