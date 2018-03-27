package scanners

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Creator is the func stub that defines how to instantiate an image scanner
type Creator func(scanner *v1.ImageIntegration) (ImageScanner, error)

// Registry maps a particular image scanner to the func that can create it
var Registry = map[string]Creator{}

// CreateScanner checks to make sure the integration exists and then tries to generate a new Scanner
// returns an error if the creation was unsuccessful
func CreateScanner(source *v1.ImageIntegration) (ImageScanner, error) {
	creator, exists := Registry[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("Scanner with type '%s' does not exist", source.GetType())
	}
	return creator(source)
}
