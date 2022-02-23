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

var (
	_ types.ImageScanner  = (*imageScannerWithDataSource)(nil)
	_ types.ScannerGetter = (*imageScannerWithDataSource)(nil)
)

type imageScannerWithDataSource struct {
	types.Scanner
	datasource *storage.DataSource
}

func (i *imageScannerWithDataSource) GetScanner() types.Scanner {
	return i.Scanner
}

func (i *imageScannerWithDataSource) DataSource() *storage.DataSource {
	return i.datasource
}

func (e *factoryImpl) CreateScanner(source *storage.ImageIntegration) (types.ImageScanner, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("scanner with type %q does not exist", source.GetType())
	}
	scanner, err := creator(source)
	if err != nil {
		return nil, err
	}
	return &imageScannerWithDataSource{
		Scanner: scanner,
		datasource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
	}, nil
}
