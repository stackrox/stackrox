package edges

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// EdgeID is a synthetic ID generated for a parent child relationship.
type EdgeID struct {
	ParentID string
	ChildID  string
}

// FromString reads a EdgeID from string form.
func FromString(str string) (EdgeID, error) {
	nameAndVersionEncoded := strings.Split(str, ":")
	if len(nameAndVersionEncoded) != 2 {
		return EdgeID{}, fmt.Errorf("invalid id: %s", str)
	}
	parentID, err := base64.URLEncoding.DecodeString(nameAndVersionEncoded[0])
	if err != nil {
		return EdgeID{}, err
	}
	childID, err := base64.URLEncoding.DecodeString(nameAndVersionEncoded[1])
	if err != nil {
		return EdgeID{}, err
	}
	return EdgeID{ParentID: string(parentID), ChildID: string(childID)}, nil
}

// ToString serializes the EdgeID to a string.
func (cID EdgeID) ToString() string {
	nameEncoded := base64.URLEncoding.EncodeToString([]byte(cID.ParentID))
	versionEncoded := base64.URLEncoding.EncodeToString([]byte(cID.ChildID))
	return fmt.Sprintf("%s:%s", nameEncoded, versionEncoded)
}
