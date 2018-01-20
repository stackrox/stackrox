package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

type scannerStore struct {
	db.ScannerStorage
}

func newScannerStore(persistent db.ScannerStorage) *scannerStore {
	return &scannerStore{
		ScannerStorage: persistent,
	}
}

// GetScanners returns a slice of scanners based on the request
func (s *scannerStore) GetScanners(request *v1.GetScannersRequest) ([]*v1.Scanner, error) {
	scanners, err := s.ScannerStorage.GetScanners(request)
	if err != nil {
		return nil, err
	}
	scannerSlice := scanners[:0]
	for _, scanner := range scanners {
		if len(request.GetCluster()) != 0 && !sliceContains(scanner.GetClusters(), request.GetCluster()) {
			continue
		}
		scannerSlice = append(scannerSlice, scanner)
	}
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
