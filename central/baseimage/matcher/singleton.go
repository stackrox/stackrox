package matcher

import baseImageDatastore "github.com/stackrox/rox/central/baseimage/datastore"

func Singleton() Matcher {
	return New(baseImageDatastore.Singleton())
}
