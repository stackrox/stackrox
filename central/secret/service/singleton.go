package service

import (
	"fmt"
	"sync"

	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	globalindex "bitbucket.org/stack-rox/apollo/central/globalindex/singletons"
	"bitbucket.org/stack-rox/apollo/central/secret/index"
	"bitbucket.org/stack-rox/apollo/central/secret/search"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

var (
	once sync.Once

	storage store.Store

	as Service
)

func initialize() {
	// Register store buckets.
	storage = store.New(globaldb.GetGlobalDB())

	// load all of the secrets from the db.
	secrets, err := storage.GetAllSecrets()
	if err != nil {
		panic(err)
	}

	// index them.
	for _, secret := range secrets {
		// Load the secrets relationships.
		relationship, exists, err := storage.GetRelationship(secret.GetId())
		if err != nil {
			panic(err)
		} else if !exists {
			panic(fmt.Sprintf("secret is missing relationship: %s", secret.GetId()))
		}

		// Index the secret and the relationship.
		err = index.SecretAndRelationship(globalindex.GetGlobalIndex(), &v1.SecretAndRelationship{
			Secret:       secret,
			Relationship: relationship,
		})
		if err != nil {
			panic(err)
		}
	}

	as = New(storage, globalindex.GetGlobalIndex())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}

// ParsedSearchRequestHandler provides a function to take a search request and return search results.
func ParsedSearchRequestHandler() func(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	once.Do(initialize)
	return func(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
		return search.ParsedSearchRequestWrapper{ParsedSearchRequest: request}.ToSearchResults(storage, globalindex.GetGlobalIndex())
	}
}
