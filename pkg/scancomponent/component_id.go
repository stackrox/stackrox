package scancomponent

import (
	"encoding/base64"
	"fmt"
)

// ComponentID creates a component ID from the given name and version
func ComponentID(name, version string) string {
	nameEncoded := base64.RawURLEncoding.EncodeToString([]byte(name))
	versionEncoded := base64.RawURLEncoding.EncodeToString([]byte(version))
	return fmt.Sprintf("%s:%s", nameEncoded, versionEncoded)
}
