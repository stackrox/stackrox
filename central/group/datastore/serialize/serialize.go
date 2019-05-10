package serialize

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// PropsKey is the key function for GroupProperties objects
func PropsKey(props *storage.GroupProperties) string {
	return StringKey(props.GetAuthProviderId(), props.GetKey(), props.GetValue())
}

// StringKey is the key function for GroupProperties objects with direct input values.
func StringKey(authProviderID, attrKey, attrValue string) string {
	return fmt.Sprintf("%s:%s:%s", authProviderID, attrKey, attrValue)
}
