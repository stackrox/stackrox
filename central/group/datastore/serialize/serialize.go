package serialize

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/binenc"
)

// PropsKey is the key function for GroupProperties objects
func PropsKey(props *storage.GroupProperties) []byte {
	return BytesKey(props.GetAuthProviderId(), props.GetKey(), props.GetValue())
}

// BytesKey is the key function for GroupProperties objects with direct input values, returning the key as a byte slice.
func BytesKey(authProviderID, attrKey, attrValue string) []byte {
	return binenc.EncodeBytesList([]byte(authProviderID), []byte(attrKey), []byte(attrValue))
}

// StringKey is the key function for GroupProperties objects with direct input values, returning the key as a string.
func StringKey(authProviderID, attrKey, attrValue string) string {
	return string(BytesKey(authProviderID, attrKey, attrValue))
}
