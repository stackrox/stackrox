package scanners

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
)

var _ Factory = (*factoryImpl)(nil)

type factoryImpl struct {
	creators map[string]Creator
}

var _ types.ImageScannerWithDataSource = (*imageScannerWithDataSource)(nil)

type imageScannerWithDataSource struct {
	scanner    types.Scanner
	datasource *storage.DataSource
}

func (i *imageScannerWithDataSource) GetScanner() types.Scanner {
	return i.scanner
}

func (i *imageScannerWithDataSource) DataSource() *storage.DataSource {
	return i.datasource
}

func (e *factoryImpl) CreateScanner(source *storage.ImageIntegration) (types.ImageScannerWithDataSource, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("scanner with type %q does not exist", source.GetType())
	}
	scanner, err := creator(source)
	if err != nil {
		return nil, err
	}
	return &imageScannerWithDataSource{
		scanner: scanner,
		datasource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
	}, nil
}
