package sources

import (
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
)

// ImageIntegration is a wrapper around the v1.ImageIntegration object that contains the created forms of the plugins
type ImageIntegration struct {
	*v1.ImageIntegration
	Registry registries.ImageRegistry
	Scanner  scanners.ImageScanner
}

// NewImageIntegration takes a v1.ImageIntegration and returns an image integration that has created the inputs
func NewImageIntegration(protoSource *v1.ImageIntegration) (*ImageIntegration, error) {
	if err := validateCommonFields(protoSource); err != nil {
		return nil, err
	}
	sortCategories(protoSource)
	integration := &ImageIntegration{
		ImageIntegration: protoSource,
	}
	var err error
	for _, category := range protoSource.GetCategories() {
		switch category {
		case v1.ImageIntegrationCategory_REGISTRY:
			integration.Registry, err = registries.CreateRegistry(protoSource)
			if err != nil {
				return nil, err
			}
		case v1.ImageIntegrationCategory_SCANNER:
			integration.Scanner, err = scanners.CreateScanner(protoSource)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("Source category '%s' has not been implemented", category)
		}
	}
	return integration, nil
}

func validateCommonFields(source *v1.ImageIntegration) error {
	errorList := errorhelpers.NewErrorList("Validation")
	if source.GetName() == "" {
		errorList.AddString("Source name must be defined")
	}
	if source.GetType() == "" {
		errorList.AddString("Source type must be defined")
	}
	if len(source.GetCategories()) == 0 {
		errorList.AddString("At least one category must be defined")
	}
	return errorList.ToError()
}

var categoryOrder = map[v1.ImageIntegrationCategory]int{
	v1.ImageIntegrationCategory_REGISTRY: 0,
	v1.ImageIntegrationCategory_SCANNER:  1,
}

func sortCategories(request *v1.ImageIntegration) {
	sort.SliceStable(request.GetCategories(), func(i, j int) bool {
		return categoryOrder[request.GetCategories()[i]] < categoryOrder[request.GetCategories()[j]]
	})
}

// Test iterates over the categories and test each of the inputs
func (ds *ImageIntegration) Test() error {
	for _, category := range ds.GetCategories() {
		switch category {
		case v1.ImageIntegrationCategory_REGISTRY:
			reg, err := registries.CreateRegistry(ds.ImageIntegration)
			if err != nil {
				return err
			}
			if err := reg.Test(); err != nil {
				return err
			}
		case v1.ImageIntegrationCategory_SCANNER:
			scanner, err := scanners.CreateScanner(ds.ImageIntegration)
			if err != nil {
				return err
			}
			if err := scanner.Test(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Source category '%s' has not been implemented", category)
		}
	}
	return nil
}
