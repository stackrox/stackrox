package scancomponent

import (
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log = logging.LoggerForModule()
)

// ComponentID creates a component ID from the given name and version and os.
func ComponentID(name, version, os string) string {
	return pgSearch.IDFromPks([]string{name, version, os})
}

type hashWrapper struct {
	Components storage.EmbeddedImageScanComponent `hash:"set"`
}

// ComponentIDV2 creates a component ID from the given name and version and architecture and imageID.
func ComponentIDV2(component *storage.EmbeddedImageScanComponent, imageID string) (string, error) {
	// A little future proofing here.  Just hashing the component to ensure uniqueness.  If a field is added, the data
	// will be replaced anyway.  We just need to ensure uniqueness within the scan since we tack on the imageID.
	component.SetTopCvss = nil
	log.Infof("SHREWS -- %q %q", component.GetName(), component.GetVersion())
	log.Infof("SHREWS -- %q", imageID)
	hash, err := hashstructure.Hash(component, hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
	if err != nil {
		return "", err
	}
	log.Infof("SHREWS -- %q", strconv.FormatUint(hash, 10))
	log.Infof("SHREWS -- %v", component.GetVulns())
	log.Infof("SHREWS -- %v", component)
	return pgSearch.IDFromPks([]string{component.GetName(), strconv.FormatUint(hash, 10), imageID}), nil
}
