package scancomponent

import (
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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
	//We must make a clone of the incoming object to use in our hash.  The `SetTopCvss` must be set to nil before hashing
	// as that is added by the enricher and may vary.  So we want to ignore it.  Since it is
	// a oneof we cannot simply flag it as ignore in the proto, sadly.
	clonedComponent := component.CloneVT()
	clonedComponent.SetTopCvss = nil

	hash, err := hashstructure.Hash(clonedComponent, hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
	if err != nil {
		return "", err
	}

	return pgSearch.IDFromPks([]string{component.GetName(), strconv.FormatUint(hash, 10), imageID}), nil
}
