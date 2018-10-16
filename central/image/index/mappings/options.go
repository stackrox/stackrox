package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = map[search.FieldLabel]*v1.SearchField{
	// Add the scope so that we can use this options map to search for deployment cluster data
	search.Cluster:   search.NewStringField(v1.SearchCategory_DEPLOYMENTS, "deployment.cluster_name"),
	search.Namespace: search.NewStringField(v1.SearchCategory_DEPLOYMENTS, "deployment.namespace"),
	search.Label:     search.NewMapField(v1.SearchCategory_DEPLOYMENTS, "deployment.labels"),

	search.CVE:                          search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.scan.components.vulns.cve"),
	search.CVELink:                      search.NewField(v1.SearchCategory_IMAGES, "image.scan.components.vulns.link", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.CVSS:                         search.NewStoredNumericField(v1.SearchCategory_IMAGES, "image.scan.components.vulns.cvss"),
	search.Component:                    search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.scan.components.name"),
	search.ComponentVersion:             search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.scan.components.version"),
	search.DockerfileInstructionKeyword: search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.metadata.v1.layers.instruction"),
	search.DockerfileInstructionValue:   search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.metadata.v1.layers.value"),
	search.ImageCreatedTime:             search.NewStoredTimeField(v1.SearchCategory_IMAGES, "image.metadata.v1.created.seconds"),
	search.ImageName:                    search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.name.full_name"),
	search.ImageSHA:                     search.NewField(v1.SearchCategory_IMAGES, "image.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.ImageRegistry:                search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.name.registry"),
	search.ImageRemote:                  search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.name.remote"),
	search.ImageScanTime:                search.NewStoredTimeField(v1.SearchCategory_IMAGES, "image.scan.scan_time.seconds"),
	search.ImageTag:                     search.NewStoredStringField(v1.SearchCategory_IMAGES, "image.name.tag"),
}
