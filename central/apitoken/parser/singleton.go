package parser

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	"bitbucket.org/stack-rox/apollo/central/role/store"
	"bitbucket.org/stack-rox/apollo/pkg/auth/tokenbased"
)

var (
	parser tokenbased.IdentityParser
	once   sync.Once
)

func initialize() {
	parser = New(signer.Singleton(), store.Singleton())
}

// Singleton returns the instance of the parser to use.
func Singleton() tokenbased.IdentityParser {
	once.Do(initialize)
	return parser
}
