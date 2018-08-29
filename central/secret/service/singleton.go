package service

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/search/transform"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	// load all of the secrets from the db.
	secrets, err := store.Singleton().GetAllSecrets()
	if err != nil {
		panic(err)
	}

	// index them.
	for _, secret := range secrets {
		// Load the secrets relationships.
		relationship, exists, err := store.Singleton().GetRelationship(secret.GetId())
		if err != nil {
			panic(err)
		} else if !exists {
			panic(fmt.Sprintf("secret is missing relationship: %s", secret.GetId()))
		}

		// Index the secret and the relationship.
		err = index.Singleton().SecretAndRelationship(&v1.SecretAndRelationship{
			Secret:       secret,
			Relationship: relationship,
		})
		if err != nil {
			panic(err)
		}
	}

	as = New(store.Singleton(), search.Singleton())
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
