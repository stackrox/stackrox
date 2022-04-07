package scancomponent

import (
	"encoding/base64"
	"fmt"

	"github.com/stackrox/rox/pkg/features"
)

// ComponentID creates a component ID from the given name and version (and os if postgres is enabled).
func ComponentID(name, version, os string) string {
	if features.PostgresDatastore.Enabled() {
		return fmt.Sprintf("%s:%s:%s", name, version, os)
	}
	nameEncoded := base64.RawURLEncoding.EncodeToString([]byte(name))
	versionEncoded := base64.RawURLEncoding.EncodeToString([]byte(version))
	return fmt.Sprintf("%s:%s", nameEncoded, versionEncoded)
}
