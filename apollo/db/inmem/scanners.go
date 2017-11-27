package inmem

import "bitbucket.org/stack-rox/apollo/apollo/scanners/types"

// AddScanner adds a scanner
func (i *InMemoryStore) AddScanner(name string, scanner types.ImageScanner) {
	i.scanMutex.Lock()
	defer i.scanMutex.Unlock()
	i.scanners[name] = scanner
}

// RemoveScanner removes a scanner
func (i *InMemoryStore) RemoveScanner(name string) {
	i.scanMutex.Lock()
	defer i.scanMutex.Unlock()
	delete(i.scanners, name)
}

// GetScanners retrieves all scanners from the db
func (i *InMemoryStore) GetScanners() map[string]types.ImageScanner {
	i.scanMutex.Lock()
	defer i.scanMutex.Unlock()
	return i.scanners
}
