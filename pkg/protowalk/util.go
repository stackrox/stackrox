package protowalk

import (
	"strings"

	"github.com/stackrox/rox/pkg/stringutils"
)

// protoTagValue extracts a value with the given key from a protobuf struct tag.
func protoTagValue(protoTag string, key string) string {
	elems := strings.Split(protoTag, ",")
	keyPrefix := key + "="
	for _, e := range elems {
		elem := e
		if stringutils.ConsumePrefix(&elem, keyPrefix) {
			return elem
		}
	}
	return ""
}
