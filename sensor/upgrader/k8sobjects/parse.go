package k8sobjects

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ParseRef parses a string representation of an object reference, in the format `<kind>:<group/version>:[<namespace>/]<name>`.
func ParseRef(str string) (ObjectRef, error) {
	parts := strings.Split(str, ":")
	if len(parts) != 3 {
		return ObjectRef{}, errors.Errorf("unexpected number of colons: %d, expected %d", len(parts)-1, 2)
	}

	var ref ObjectRef
	ref.GVK.Kind = parts[0]
	gv, err := schema.ParseGroupVersion(parts[1])
	if err != nil {
		return ObjectRef{}, errors.Wrap(err, "parsing GroupVersion part")
	}
	ref.GVK.Group = gv.Group
	ref.GVK.Version = gv.Version

	nameParts := strings.Split(parts[2], "/")
	if len(nameParts) > 2 {
		return ObjectRef{}, errors.Errorf("too many slashes in name part: %d, expected at most %d", len(nameParts)-1, 1)
	}
	if len(nameParts) == 2 {
		ref.Namespace = nameParts[0]
		nameParts = nameParts[1:]
	}
	ref.Name = nameParts[0]

	return ref, nil
}
