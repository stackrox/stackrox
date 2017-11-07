package mockscanner

import (
	"bitbucket.org/stack-rox/apollo/apollo/scanners"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type mockScanner struct{}

func new(_ map[string]string) (scannerTypes.ImageScanner, error) {
	return nil, nil
}

// GetScan takes in an id and returns the image scan for that id if applicable
func (m *mockScanner) GetScan(id string) (*v1.ImageScan, error) {
	return nil, nil
}

// Scan initiates a scan of the passed id
func (m *mockScanner) Scan(id string) error {
	return nil
}

func init() {
	scanners.Registry["mockscanner"] = new
}
