package imagecomponent

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// ComponentID is a synthetic ID for component objects composed of the name and version of the component.
type ComponentID struct {
	Name    string
	Version string
}

// FromString reads a ComponentID from string form.
func FromString(str string) (ComponentID, error) {
	nameAndVersionEncoded := strings.Split(str, ":")
	if len(nameAndVersionEncoded) != 2 {
		return ComponentID{}, fmt.Errorf("invalid id: %s", str)
	}
	name, err := base64.RawURLEncoding.DecodeString(nameAndVersionEncoded[0])
	if err != nil {
		return ComponentID{}, err
	}
	version, err := base64.RawURLEncoding.DecodeString(nameAndVersionEncoded[1])
	if err != nil {
		return ComponentID{}, err
	}
	return ComponentID{Name: string(name), Version: string(version)}, nil
}

// ToString serializes the ComponentID to a url string.
func (cID ComponentID) ToString() string {
	nameEncoded := base64.RawURLEncoding.EncodeToString([]byte(cID.Name))
	versionEncoded := base64.RawURLEncoding.EncodeToString([]byte(cID.Version))
	return fmt.Sprintf("%s:%s", nameEncoded, versionEncoded)
}
