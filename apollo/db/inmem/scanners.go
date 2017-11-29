package inmem

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
)

type scannerStore struct {
	scanners  map[string]scannerTypes.ImageScanner
	scanMutex sync.Mutex

	persistent db.Storage
}

func newScannerStore(persistent db.Storage) *scannerStore {
	return &scannerStore{
		scanners:   make(map[string]scannerTypes.ImageScanner),
		persistent: persistent,
	}
}

// AddScanner adds a scanner
func (s *scannerStore) AddScanner(name string, scanner scannerTypes.ImageScanner) {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()
	s.scanners[name] = scanner
}

// RemoveScanner removes a scanner
func (s *scannerStore) RemoveScanner(name string) {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()
	delete(s.scanners, name)
}

// GetScanners retrieves all scanners from the db
func (s *scannerStore) GetScanners() map[string]scannerTypes.ImageScanner {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()
	return s.scanners
}
