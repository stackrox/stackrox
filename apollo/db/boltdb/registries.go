package boltdb

import "bitbucket.org/stack-rox/apollo/apollo/registries/types"

// AddRegistry upserts a registry into bolt
func (b *BoltDB) AddRegistry(name string, registry types.ImageRegistry) {
	panic("implement me")
}

// RemoveRegistry removes a registry from bolt
func (b *BoltDB) RemoveRegistry(name string) {
	panic("implement me")
}

// GetRegistries retrieves registries from bolt
func (b *BoltDB) GetRegistries() map[string]types.ImageRegistry {
	panic("implement me")
}
