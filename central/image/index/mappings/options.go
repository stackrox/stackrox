package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = map[string]*v1.SearchField{
	search.CVE:                          search.NewStringField("image.scan.components.vulns.cve"),
	search.CVSS:                         search.NewNumericField("image.scan.components.vulns.cvss"),
	search.Component:                    search.NewStringField("image.scan.components.name"),
	search.DockerfileInstructionKeyword: search.NewStringField("image.metadata.layers.instruction"),
	search.DockerfileInstructionValue:   search.NewStringField("image.metadata.layers.value"),
	search.ImageName:                    search.NewStringField("image.name.full_name"),
	search.ImageSHA:                     search.NewField("image.name.sha", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.ImageRegistry:                search.NewStringField("image.name.registry"),
	search.ImageRemote:                  search.NewStringField("image.name.remote"),
	search.ImageTag:                     search.NewStringField("image.name.tag"),
}
