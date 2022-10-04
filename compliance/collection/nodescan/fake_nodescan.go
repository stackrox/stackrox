package nodescan

import (
	"fmt"

	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/pkg/analyzer/nodes"
	"github.com/stackrox/scanner/pkg/component"
)

func FakeCollect() error {
	// The real nodes.Analyze() call returns a Components pointer
	components := nodes.Components{
		OSNamespace: &database.Namespace{
			Name:          "Fake RHEL",
			VersionFormat: "42",
		},
		OSComponents:            []database.FeatureVersion{},
		CertifiedRHELComponents: nil,
		LanguageComponents:      []*component.Component{},
	}

	fmt.Printf("components: %v", components)
	return nil
}
