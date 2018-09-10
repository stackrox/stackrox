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

	search.CVE:                          search.NewStringField(v1.SearchCategory_IMAGES, "image.scan.components.vulns.cve"),
	search.CVSS:                         search.NewNumericField(v1.SearchCategory_IMAGES, "image.scan.components.vulns.cvss"),
	search.Component:                    search.NewStringField(v1.SearchCategory_IMAGES, "image.scan.components.name"),
	search.DockerfileInstructionKeyword: search.NewStringField(v1.SearchCategory_IMAGES, "image.metadata.layers.instruction"),
	search.DockerfileInstructionValue:   search.NewStringField(v1.SearchCategory_IMAGES, "image.metadata.layers.value"),
	search.ImageCreatedTime:             search.NewTimeField(v1.SearchCategory_IMAGES, "image.metadata.created.seconds"),
	search.ImageName:                    search.NewStringField(v1.SearchCategory_IMAGES, "image.name.full_name"),
	search.ImageSHA:                     search.NewField(v1.SearchCategory_IMAGES, "image.name.sha", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.ImageRegistry:                search.NewStringField(v1.SearchCategory_IMAGES, "image.name.registry"),
	search.ImageRemote:                  search.NewStringField(v1.SearchCategory_IMAGES, "image.name.remote"),
	search.ImageScanTime:                search.NewTimeField(v1.SearchCategory_IMAGES, "image.scan.scan_time.seconds"),
	search.ImageTag:                     search.NewField(v1.SearchCategory_IMAGES, "image.name.tag", v1.SearchDataType_SEARCH_STRING, search.OptionStore),
}
