package service

import (
	"github.com/stackrox/rox/central/aiworkload/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	as   Service
)

func initialize() {
	as = New(datastore.Singleton())
}

func Singleton() Service {
	once.Do(initialize)
	return as
}
