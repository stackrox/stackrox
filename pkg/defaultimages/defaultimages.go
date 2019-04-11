package defaultimages

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
)

// Constants used to generate and test registry defaults
const (
	Collector  = "collector"
	Monitoring = "monitoring"

	ProdMainRegistry      = "stackrox.io"
	ProdRedHatRegistry    = "registry.connect.redhat.com"
	ProdCollectorRegistry = "collector.stackrox.io"
)

// GenerateNamedImageFromMainImage given a main image repository, a version tag, and an image name, generate the default repository for the given image
func GenerateNamedImageFromMainImage(mainImageName *storage.ImageName, tag string, name string) *storage.ImageName {
	// Populate the tag
	newImageName := &storage.ImageName{
		Tag: tag,
	}
	// Populate Registry
	newImageName.Registry = mainImageName.GetRegistry()
	if mainImageName.GetRegistry() == ProdMainRegistry || mainImageName.GetRegistry() == ProdRedHatRegistry {
		newImageName.Registry = getRegistry(name)
	}
	// Populate Remote
	// This handles the case where there is no namespace. e.g. stackrox.io/NAME:latest
	if slashIdx := strings.Index(mainImageName.GetRemote(), "/"); slashIdx == -1 {
		newImageName.Remote = name
	} else {
		newImageName.Remote = mainImageName.GetRemote()[:slashIdx] + "/" + name
	}
	// Populate FullName
	utils.NormalizeImageFullNameNoSha(newImageName)
	return newImageName
}

func getRegistry(name string) string {
	if name == Collector {
		return ProdCollectorRegistry
	}
	return ProdMainRegistry
}
