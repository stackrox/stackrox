package boltdb

import "bitbucket.org/stack-rox/apollo/apollo/scanners/types"

// AddScanner upserts a scaanner into bolt
func (b *BoltDB) AddScanner(name string, scanner types.ImageScanner) {
	panic("implement me")
}

// RemoveScanner removes a scanner from bolt
func (b *BoltDB) RemoveScanner(name string) {
	panic("implement me")
}

// GetScanners retrieves scanners from bolt
func (b *BoltDB) GetScanners() map[string]types.ImageScanner {
	panic("implement me")
}
