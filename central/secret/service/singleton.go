package service

import (
	"sync"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/search/transform"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	once sync.Once
	as   Service
)

func initialize() {
	as = New(store.Singleton(), search.Singleton(), datastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}

// QueryHandler provides a function to take a search request and return search results.
func QueryHandler() func(q *v1.Query) ([]*v1.SearchResult, error) {
	once.Do(initialize)
	return func(q *v1.Query) ([]*v1.SearchResult, error) {
		return transform.ProtoQueryWrapper{Query: q}.ToSearchResults(store.Singleton(), globalindex.GetGlobalIndex())
	}
}
