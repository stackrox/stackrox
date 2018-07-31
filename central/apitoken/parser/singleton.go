package parser

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/apitoken/cachedstore"
	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	"bitbucket.org/stack-rox/apollo/central/role/store"
)

var (
	parser Parser
	once   sync.Once
)

func initialize() {
	parser = New(signer.Singleton(), store.Singleton(), cachedstore.Singleton())
}

// Singleton returns the instance of the parser to use.
func Singleton() Parser {
	once.Do(initialize)
	return parser
}
