package parser

import (
	"sync"

	"github.com/stackrox/rox/central/apitoken/cachedstore"
	"github.com/stackrox/rox/central/apitoken/signer"
	"github.com/stackrox/rox/central/role/store"
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
