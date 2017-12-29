package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/proto"
)

type scannerStore struct {
	scanners     map[string]*v1.Scanner
	scannerMutex sync.Mutex

	persistent db.ScannerStorage
}

func newScannerStore(persistent db.ScannerStorage) *scannerStore {
	return &scannerStore{
		scanners:   make(map[string]*v1.Scanner),
		persistent: persistent,
	}
}

func (s *scannerStore) clone(scanner *v1.Scanner) *v1.Scanner {
	return proto.Clone(scanner).(*v1.Scanner)
}

func (s *scannerStore) loadFromPersistent() error {
	s.scannerMutex.Lock()
	defer s.scannerMutex.Unlock()
	scanners, err := s.persistent.GetScanners(&v1.GetScannersRequest{})
	if err != nil {
		return err
	}
	for _, scanner := range scanners {
		s.scanners[scanner.Name] = scanner
	}
	return nil
}

// GetScanner returns a scanner, if it exists or an error based on the name parameter
func (s *scannerStore) GetScanner(name string) (scanner *v1.Scanner, exists bool, err error) {
	s.scannerMutex.Lock()
	defer s.scannerMutex.Unlock()
	scanner, exists = s.scanners[name]
	return s.clone(scanner), exists, nil
}

// GetScanners returns a slice of scanners based on the request
func (s *scannerStore) GetScanners(request *v1.GetScannersRequest) ([]*v1.Scanner, error) {
	s.scannerMutex.Lock()
	defer s.scannerMutex.Unlock()
	scannerSlice := make([]*v1.Scanner, 0, len(s.scanners))
	for _, scanner := range s.scanners {
		if len(request.GetCluster()) != 0 && !sliceContains(scanner.GetClusters(), request.GetCluster()) {
			continue
		}
		scannerSlice = append(scannerSlice, s.clone(scanner))
	}
	sort.SliceStable(scannerSlice, func(i, j int) bool { return scannerSlice[i].Name < scannerSlice[j].Name })
	return scannerSlice, nil
}

func sliceContains(slice []string, wanted string) bool {
	for _, val := range slice {
		if val == wanted {
			return true
		}
	}
	return false
}

func (s *scannerStore) upsertScanner(scanner *v1.Scanner) {
	s.scanners[scanner.Name] = s.clone(scanner)
}

// AddScanner upserts a scanner
func (s *scannerStore) AddScanner(scanner *v1.Scanner) error {
	s.scannerMutex.Lock()
	defer s.scannerMutex.Unlock()
	if _, exists := s.scanners[scanner.Name]; exists {
		return fmt.Errorf("Scanner with name %v already exists", scanner.Name)
	}
	if err := s.persistent.AddScanner(scanner); err != nil {
		return err
	}
	s.upsertScanner(scanner)
	return nil
}

// UpdateScanner upserts a scanner
func (s *scannerStore) UpdateScanner(scanner *v1.Scanner) error {
	s.scannerMutex.Lock()
	defer s.scannerMutex.Unlock()
	if err := s.persistent.UpdateScanner(scanner); err != nil {
		return err
	}
	s.upsertScanner(scanner)
	return nil
}

// RemoveScanner removes a scanner
func (s *scannerStore) RemoveScanner(name string) error {
	s.scannerMutex.Lock()
	defer s.scannerMutex.Unlock()
	if err := s.persistent.RemoveScanner(name); err != nil {
		return err
	}
	delete(s.scanners, name)
	return nil
}
